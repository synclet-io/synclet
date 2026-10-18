// Command demo-seed fills a running Synclet with a demo workspace: a user,
// four connections on mock data and a few finished sync runs. Every name and
// record is made up; the sources are airbyte/source-faker and a PostgreSQL
// database the command creates and fills with generated rows.
//
//	go run ./scripts/demo-seed -url http://localhost:8080
//
// See scripts/demo-seed/README.md for the flags and what gets created.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/multierr"
	"google.golang.org/protobuf/types/known/structpb"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	workspacev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1/workspacev1connect"
)

const (
	fakerImage      = "airbyte/source-faker"
	fakerTag        = "6.2.38"
	sourcePGImage   = "airbyte/source-postgres"
	sourcePGTag     = "3.8.7"
	destPGImage     = "airbyte/destination-postgres"
	destPGTag       = "3.0.23"
	inventoryDB     = "demo_inventory"
	warehouseDB     = "demo_warehouse"
	partnerFeedDB   = "demo_partner_feed"
	hourly          = "0 * * * *"
	jobWaitTimeout  = 15 * time.Minute
	jobPollInterval = 2 * time.Second
)

type options struct {
	url             string
	email           string
	password        string
	userName        string
	workspaceName   string
	pgDSN           string
	connectorPGHost string
	connectorPGPort int
	runs            int
}

func main() {
	var opts options
	flag.StringVar(&opts.url, "url", "http://localhost:8080", "Synclet base URL")
	flag.StringVar(&opts.email, "email", "demo@example.com", "demo user email (registered, or logged in when it exists)")
	flag.StringVar(&opts.password, "password", "demo-password", "demo user password")
	flag.StringVar(&opts.userName, "name", "Demo User", "demo user display name")
	flag.StringVar(&opts.workspaceName, "workspace", "Demo Shop", "name given to the demo user's workspace")
	flag.StringVar(&opts.pgDSN, "pg-dsn", "",
		"PostgreSQL the command drops and recreates the demo_* databases in, as reached from this machine (required)")
	flag.StringVar(&opts.connectorPGHost, "connector-pg-host", "host.docker.internal",
		"host of that PostgreSQL as reached from connector containers")
	flag.IntVar(&opts.connectorPGPort, "connector-pg-port", 0,
		"port of that PostgreSQL as reached from connector containers (default: the port of -pg-dsn)")
	flag.IntVar(&opts.runs, "runs", 2, "sync runs per connection")
	flag.Parse()

	if opts.pgDSN == "" {
		flag.Usage()
		log.Fatal("-pg-dsn is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	err := run(ctx, opts)

	cancel()

	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, opts options) error {
	demoPG, err := newDemoPostgres(opts)
	if err != nil {
		return err
	}

	api := newClient(opts.url)
	if err := api.login(ctx, opts); err != nil {
		return err
	}

	log.Printf("workspace %q (%s)", opts.workspaceName, api.workspaceID)

	conns, err := api.connection.ListConnections(ctx, authed(api, &pipelinev1.ListConnectionsRequest{WorkspaceId: api.workspaceID}))
	if err != nil {
		return fmt.Errorf("listing connections: %w", err)
	}

	if n := len(conns.Msg.GetConnections()); n > 0 {
		return fmt.Errorf("workspace already has %d connections; seed a fresh Synclet database", n)
	}

	if err := demoPG.prepare(ctx); err != nil {
		return err
	}

	connectors, err := api.addConnectors(ctx)
	if err != nil {
		return err
	}

	storefront, err := api.createSource(ctx, "Storefront (Faker)", connectors.faker, map[string]any{
		"count": 500, "seed": 7, "records_per_slice": 1000, "always_updated": false, "parallelism": 1,
	})
	if err != nil {
		return err
	}

	inventory, err := api.createSource(ctx, "Inventory DB", connectors.sourcePG, demoPG.sourceConfig(inventoryDB))
	if err != nil {
		return err
	}

	partnerFeed, err := api.createSource(ctx, "Partner feed", connectors.sourcePG, demoPG.sourceConfig(partnerFeedDB))
	if err != nil {
		return err
	}

	warehouse, err := api.createDestination(ctx, "Analytics warehouse", connectors.destPG, demoPG.destinationConfig(warehouseDB))
	if err != nil {
		return err
	}

	for _, sourceID := range []string{storefront, inventory, partnerFeed} {
		if err := api.discover(ctx, sourceID); err != nil {
			return err
		}
	}

	plans := []connectionPlan{
		{
			name: "Storefront users → Warehouse", sourceID: storefront, namespace: "storefront",
			streams: []*pipelinev1.ConfiguredStream{incremental("users", "", "updated_at")},
		},
		{
			name: "Storefront orders → Warehouse", sourceID: storefront, namespace: "orders",
			streams: []*pipelinev1.ConfiguredStream{
				incremental("purchases", "", "updated_at"),
				incremental("products", "", "updated_at"),
			},
		},
		{
			name: "Inventory DB → Warehouse", sourceID: inventory, namespace: "inventory",
			streams: []*pipelinev1.ConfiguredStream{
				incremental("stock_items", "public", "id"),
				incremental("warehouses", "public", "id"),
			},
		},
	}

	var syncIDs []string

	for _, p := range plans {
		id, err := api.createConnection(ctx, p, warehouse)
		if err != nil {
			return err
		}

		syncIDs = append(syncIDs, id)
	}

	partnerID, err := api.createConnection(ctx, connectionPlan{
		name: "Partner feed → Warehouse", sourceID: partnerFeed, namespace: "partner", maxAttempts: 1,
		streams: []*pipelinev1.ConfiguredStream{incremental("shipments", "public", "id")},
	}, warehouse)
	if err != nil {
		return err
	}

	for run := 1; run <= opts.runs; run++ {
		for _, id := range syncIDs {
			if err := api.syncAndWait(ctx, id, run); err != nil {
				return err
			}
		}

		if run == 1 {
			if err := demoPG.addInventoryRows(ctx); err != nil {
				return err
			}
		}
	}

	// The partner feed database goes away after setup, so its connection
	// fails on every run and the dashboard shows a failure.
	if err := demoPG.dropDatabase(ctx, partnerFeedDB); err != nil {
		return err
	}

	if err := api.syncAndWait(ctx, partnerID, 1); err != nil {
		return err
	}

	log.Printf("done: sign in at %s as %s", opts.url, opts.email)

	return nil
}

type demoPostgres struct {
	adminDSN *url.URL
	host     string
	port     int
	user     string
	password string
}

func newDemoPostgres(opts options) (*demoPostgres, error) {
	dsn, err := url.Parse(opts.pgDSN)
	if err != nil {
		return nil, fmt.Errorf("parsing -pg-dsn: %w", err)
	}

	port := opts.connectorPGPort
	if port == 0 {
		port = 5432
		if p := dsn.Port(); p != "" {
			if port, err = strconv.Atoi(p); err != nil {
				return nil, fmt.Errorf("parsing port of -pg-dsn: %w", err)
			}
		}
	}

	password, _ := dsn.User.Password()

	return &demoPostgres{adminDSN: dsn, host: opts.connectorPGHost, port: port, user: dsn.User.Username(), password: password}, nil
}

func (p *demoPostgres) open(database string) (*sql.DB, error) {
	u := *p.adminDSN
	u.Path = "/" + database

	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", database, err)
	}

	return db, nil
}

// prepare recreates the demo databases: an inventory source with generated
// rows and an empty warehouse for the destination.
func (p *demoPostgres) prepare(ctx context.Context) (rerr error) {
	admin, err := p.open(p.adminDSN.Path[1:])
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(admin))

	for _, name := range []string{inventoryDB, warehouseDB, partnerFeedDB} {
		if _, err := admin.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name)); err != nil {
			return fmt.Errorf("dropping %s: %w", name, err)
		}

		if _, err := admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %q`, name)); err != nil {
			return fmt.Errorf("creating %s: %w", name, err)
		}
	}

	partner, err := p.open(partnerFeedDB)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(partner))

	_, err = partner.ExecContext(ctx, `
		CREATE TABLE public.shipments (id int PRIMARY KEY, tracking_code text NOT NULL, shipped_at timestamptz NOT NULL DEFAULT now());
		INSERT INTO public.shipments (id, tracking_code) SELECT g, 'TRK-' || lpad(g::text, 6, '0') FROM generate_series(1, 80) g;`)
	if err != nil {
		return fmt.Errorf("filling %s: %w", partnerFeedDB, err)
	}

	inv, err := p.open(inventoryDB)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(inv))

	_, err = inv.ExecContext(ctx, `
		CREATE TABLE public.warehouses (
			id int PRIMARY KEY,
			code text NOT NULL,
			city text NOT NULL
		);
		INSERT INTO public.warehouses (id, code, city) VALUES
			(1, 'WH-NORTH', 'Riverton'), (2, 'WH-SOUTH', 'Lakeview'), (3, 'WH-EAST', 'Cedar Falls'),
			(4, 'WH-WEST', 'Pinecrest'), (5, 'WH-HUB', 'Maple Junction');
		CREATE TABLE public.stock_items (
			id int PRIMARY KEY,
			sku text NOT NULL,
			warehouse_id int NOT NULL REFERENCES public.warehouses (id),
			quantity int NOT NULL,
			updated_at timestamptz NOT NULL DEFAULT now()
		);
		INSERT INTO public.stock_items (id, sku, warehouse_id, quantity)
			SELECT g, 'SKU-' || lpad(g::text, 5, '0'), 1 + g % 5, (g * 37) % 400
			FROM generate_series(1, 1200) g;`)
	if err != nil {
		return fmt.Errorf("filling %s: %w", inventoryDB, err)
	}

	return nil
}

func (p *demoPostgres) dropDatabase(ctx context.Context, name string) (rerr error) {
	admin, err := p.open(p.adminDSN.Path[1:])
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(admin))

	if _, err := admin.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %q WITH (FORCE)`, name)); err != nil {
		return fmt.Errorf("dropping %s: %w", name, err)
	}

	return nil
}

// addInventoryRows adds rows between runs so the second incremental run reads new records.
func (p *demoPostgres) addInventoryRows(ctx context.Context) (rerr error) {
	inv, err := p.open(inventoryDB)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(inv))

	_, err = inv.ExecContext(ctx, `INSERT INTO public.stock_items (id, sku, warehouse_id, quantity)
		SELECT g, 'SKU-' || lpad(g::text, 5, '0'), 1 + g % 5, (g * 37) % 400
		FROM generate_series(1201, 1350) g`)
	if err != nil {
		return fmt.Errorf("adding inventory rows: %w", err)
	}

	return nil
}

func (p *demoPostgres) sourceConfig(database string) map[string]any {
	return map[string]any{
		"host": p.host, "port": p.port, "database": database, "schemas": []any{"public"},
		"username": p.user, "password": p.password,
		"ssl_mode":           map[string]any{"mode": "disable"},
		"replication_method": map[string]any{"method": "Standard"},
		"tunnel_method":      map[string]any{"tunnel_method": "NO_TUNNEL"},
	}
}

func (p *demoPostgres) destinationConfig(database string) map[string]any {
	return map[string]any{
		"host": p.host, "port": p.port, "database": database, "schema": "public",
		"username": p.user, "password": p.password,
		"ssl_mode":      map[string]any{"mode": "disable"},
		"tunnel_method": map[string]any{"tunnel_method": "NO_TUNNEL"},
	}
}

type client struct {
	auth        authv1connect.AuthServiceClient
	workspace   workspacev1connect.WorkspaceServiceClient
	registry    registryv1connect.ConnectorRegistryServiceClient
	source      pipelinev1connect.SourceServiceClient
	destination pipelinev1connect.DestinationServiceClient
	connection  pipelinev1connect.ConnectionServiceClient
	job         pipelinev1connect.JobServiceClient
	token       string
	workspaceID string
}

func newClient(baseURL string) *client {
	httpClient := http.DefaultClient

	return &client{
		auth:        authv1connect.NewAuthServiceClient(httpClient, baseURL),
		workspace:   workspacev1connect.NewWorkspaceServiceClient(httpClient, baseURL),
		registry:    registryv1connect.NewConnectorRegistryServiceClient(httpClient, baseURL),
		source:      pipelinev1connect.NewSourceServiceClient(httpClient, baseURL),
		destination: pipelinev1connect.NewDestinationServiceClient(httpClient, baseURL),
		connection:  pipelinev1connect.NewConnectionServiceClient(httpClient, baseURL),
		job:         pipelinev1connect.NewJobServiceClient(httpClient, baseURL),
	}
}

func authed[T any](c *client, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+c.token)

	if c.workspaceID != "" {
		req.Header().Set("Workspace-Id", c.workspaceID)
	}

	return req
}

func accessToken(h http.Header) string {
	for _, cookie := range (&http.Response{Header: h}).Cookies() {
		if cookie.Name == "synclet_at" {
			return cookie.Value
		}
	}

	return ""
}

// login registers the demo user, or logs in when it exists, and picks the
// user's first workspace, creating one when there is none.
func (c *client) login(ctx context.Context, opts options) error {
	reg, err := c.auth.Register(ctx, connect.NewRequest(&authv1.RegisterRequest{
		Email: opts.email, Password: opts.password, Name: opts.userName,
	}))
	if err == nil {
		c.token = accessToken(reg.Header())
	} else {
		resp, loginErr := c.auth.Login(ctx, connect.NewRequest(&authv1.LoginRequest{Email: opts.email, Password: opts.password}))
		if loginErr != nil {
			return fmt.Errorf("registering demo user: %w", errors.Join(err, loginErr))
		}

		c.token = accessToken(resp.Header())
	}

	if c.token == "" {
		return errors.New("no access token cookie in the auth response")
	}

	list, err := c.workspace.ListWorkspaces(ctx, authed(c, &workspacev1.ListWorkspacesRequest{}))
	if err != nil {
		return fmt.Errorf("listing workspaces: %w", err)
	}

	if ws := list.Msg.GetWorkspaces(); len(ws) > 0 {
		c.workspaceID = ws[0].GetWorkspace().GetId()
		if _, err := c.workspace.UpdateWorkspace(ctx, authed(c, &workspacev1.UpdateWorkspaceRequest{
			Id: c.workspaceID, Name: opts.workspaceName,
		})); err != nil {
			return fmt.Errorf("renaming workspace: %w", err)
		}

		return nil
	}

	currentUser, err := c.auth.GetCurrentUser(ctx, authed(c, &authv1.GetCurrentUserRequest{}))
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	created, err := c.workspace.CreateWorkspace(ctx, authed(c, &workspacev1.CreateWorkspaceRequest{
		Name: opts.workspaceName, OwnerUserId: currentUser.Msg.GetUser().GetId(),
	}))
	if err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	c.workspaceID = created.Msg.GetWorkspace().GetId()

	return nil
}

type demoConnectors struct {
	faker, sourcePG, destPG string
}

// addConnectors adds the pinned images, linked to the Airbyte repository when
// the server has synced one, so the UI shows their icons and docs links.
func (c *client) addConnectors(ctx context.Context) (*demoConnectors, error) {
	repos, err := c.registry.ListRepositories(ctx, authed(c, &registryv1.ListRepositoriesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("listing repositories: %w", err)
	}

	var repoID string

	for _, r := range repos.Msg.GetRepositories() {
		if strings.Contains(r.GetUrl(), "airbyte.com") {
			repoID = r.GetId()
		}
	}

	existing, err := c.registry.ListManagedConnectors(ctx, authed(c, &registryv1.ListManagedConnectorsRequest{}))
	if err != nil {
		return nil, fmt.Errorf("listing connectors: %w", err)
	}

	add := func(image, tag, name string, typ registryv1.ConnectorType) (string, error) {
		for _, mc := range existing.Msg.GetConnectors() {
			if mc.GetDockerImage() == image && mc.GetDockerTag() == tag {
				return mc.GetId(), nil
			}
		}

		log.Printf("adding connector %s:%s", image, tag)

		resp, err := c.registry.AddConnector(ctx, authed(c, &registryv1.AddConnectorRequest{
			DockerImage: image, DockerTag: tag, Name: name, ConnectorType: typ, RepositoryId: repoID,
		}))
		if err != nil {
			return "", fmt.Errorf("adding connector %s:%s: %w", image, tag, err)
		}

		return resp.Msg.GetId(), nil
	}

	var out demoConnectors
	if out.faker, err = add(fakerImage, fakerTag, "Sample Data (Faker)", registryv1.ConnectorType_CONNECTOR_TYPE_SOURCE); err != nil {
		return nil, err
	}

	if out.sourcePG, err = add(sourcePGImage, sourcePGTag, "Postgres", registryv1.ConnectorType_CONNECTOR_TYPE_SOURCE); err != nil {
		return nil, err
	}

	if out.destPG, err = add(destPGImage, destPGTag, "Postgres", registryv1.ConnectorType_CONNECTOR_TYPE_DESTINATION); err != nil {
		return nil, err
	}

	return &out, nil
}

func (c *client) createSource(ctx context.Context, name, connectorID string, config map[string]any) (string, error) {
	cfg, err := structpb.NewStruct(config)
	if err != nil {
		return "", fmt.Errorf("source %q config: %w", name, err)
	}

	resp, err := c.source.CreateSource(ctx, authed(c, &pipelinev1.CreateSourceRequest{
		WorkspaceId: c.workspaceID, Name: name, ManagedConnectorId: connectorID, Config: cfg,
	}))
	if err != nil {
		return "", fmt.Errorf("creating source %q: %w", name, err)
	}

	return resp.Msg.GetSource().GetId(), nil
}

func (c *client) createDestination(ctx context.Context, name, connectorID string, config map[string]any) (string, error) {
	cfg, err := structpb.NewStruct(config)
	if err != nil {
		return "", fmt.Errorf("destination %q config: %w", name, err)
	}

	resp, err := c.destination.CreateDestination(ctx, authed(c, &pipelinev1.CreateDestinationRequest{
		WorkspaceId: c.workspaceID, Name: name, ManagedConnectorId: connectorID, Config: cfg,
	}))
	if err != nil {
		return "", fmt.Errorf("creating destination %q: %w", name, err)
	}

	return resp.Msg.GetDestination().GetId(), nil
}

func (c *client) discover(ctx context.Context, sourceID string) error {
	if _, err := c.source.DiscoverSourceSchema(ctx, authed(c, &pipelinev1.DiscoverSourceSchemaRequest{SourceId: sourceID})); err != nil {
		return fmt.Errorf("discovering source %s: %w", sourceID, err)
	}

	return nil
}

type connectionPlan struct {
	name        string
	sourceID    string
	namespace   string
	maxAttempts int32
	streams     []*pipelinev1.ConfiguredStream
}

func incremental(stream, namespace, cursor string) *pipelinev1.ConfiguredStream {
	return &pipelinev1.ConfiguredStream{
		StreamName:          stream,
		Namespace:           namespace,
		SyncMode:            pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL,
		DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND,
		CursorField:         []string{cursor},
	}
}

func (c *client) createConnection(ctx context.Context, plan connectionPlan, destinationID string) (string, error) {
	schedule := hourly

	resp, err := c.connection.CreateConnection(ctx, authed(c, &pipelinev1.CreateConnectionRequest{
		WorkspaceId:           c.workspaceID,
		Name:                  plan.name,
		SourceId:              plan.sourceID,
		DestinationId:         destinationID,
		Schedule:              &schedule,
		SchemaChangePolicy:    pipelinev1.SchemaChangePolicy_SCHEMA_CHANGE_POLICY_PROPAGATE,
		MaxAttempts:           plan.maxAttempts,
		NamespaceDefinition:   pipelinev1.NamespaceDefinition_NAMESPACE_DEFINITION_CUSTOM,
		CustomNamespaceFormat: plan.namespace,
	}))
	if err != nil {
		return "", fmt.Errorf("creating connection %q: %w", plan.name, err)
	}

	id := resp.Msg.GetConnection().GetId()
	if len(plan.streams) > 0 {
		if _, err := c.connection.ConfigureStreams(ctx, authed(c, &pipelinev1.ConfigureStreamsRequest{
			ConnectionId: id, Streams: plan.streams,
		})); err != nil {
			return "", fmt.Errorf("configuring streams of %q: %w", plan.name, err)
		}
	}

	log.Printf("connection %q (%s)", plan.name, id)

	return id, nil
}

// syncAndWait triggers a sync and waits for it to end. A failed job is
// logged, not returned: the demo keeps a failing connection on purpose.
func (c *client) syncAndWait(ctx context.Context, connectionID string, run int) error {
	resp, err := c.job.TriggerSync(ctx, authed(c, &pipelinev1.TriggerSyncRequest{ConnectionId: connectionID}))
	if err != nil {
		return fmt.Errorf("triggering sync of %s: %w", connectionID, err)
	}

	jobID := resp.Msg.GetJob().GetId()
	deadline := time.Now().Add(jobWaitTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jobPollInterval):
		}

		got, err := c.job.GetJob(ctx, authed(c, &pipelinev1.GetJobRequest{JobId: jobID}))
		if err != nil {
			return fmt.Errorf("getting job %s: %w", jobID, err)
		}

		job := got.Msg.GetJob()
		switch job.GetStatus() {
		case pipelinev1.JobStatus_JOB_STATUS_COMPLETED, pipelinev1.JobStatus_JOB_STATUS_FAILED, pipelinev1.JobStatus_JOB_STATUS_CANCELLED:
			var records int64
			for _, a := range job.GetAttempts() {
				records += a.GetSyncStats().GetRecordsRead()
			}

			log.Printf("connection %s run %d: %s, %d records read", connectionID, run,
				strings.ToLower(strings.TrimPrefix(job.GetStatus().String(), "JOB_STATUS_")), records)

			return nil
		default:
		}
	}

	return fmt.Errorf("job %s did not finish within %s", jobID, jobWaitTimeout)
}
