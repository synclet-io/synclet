module github.com/synclet-io/synclet

go 1.25.0

require (
	connectrpc.com/connect v1.19.1
	github.com/GoAdminGroup/go-admin v1.2.26
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/caarlos0/env/v10 v10.0.0
	github.com/cespare/xxhash v1.1.0
	github.com/coreos/go-oidc/v3 v3.17.0
	github.com/docker/docker v28.5.2+incompatible
	github.com/go-pnp/go-pnp v1.1.4
	github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling v0.0.6
	github.com/go-pnp/go-pnp/connectrpc/pnprecoverconnectrpchandling v0.0.1
	github.com/go-pnp/go-pnp/healthcheck/pnphealthcheck v0.0.8
	github.com/go-pnp/go-pnp/healthcheck/pnphealthcheckgorm v0.1.5
	github.com/go-pnp/go-pnp/http/pnphttphealthcheck v0.0.23
	github.com/go-pnp/go-pnp/http/pnphttpserver v0.0.14
	github.com/go-pnp/go-pnp/http/pnphttpservercors v0.0.5
	github.com/go-pnp/go-pnp/http/pnphttpserverrecovery v0.0.4
	github.com/go-pnp/go-pnp/http/pnppromhttp v0.0.24
	github.com/go-pnp/go-pnp/logging/pnpzap v0.0.16
	github.com/go-pnp/go-pnp/pnpenv v1.0.4
	github.com/go-pnp/go-pnp/pnpjobber v1.1.1
	github.com/go-pnp/go-pnp/prometheus/pnpprometheus v0.0.13
	github.com/go-pnp/go-pnp/sql/pnpgorm v0.0.15
	github.com/go-pnp/jobber v1.3.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jackc/pgx/v5 v5.9.1
	github.com/joho/godotenv v1.5.1
	github.com/lib/pq v1.12.1
	github.com/pkg/errors v0.9.1
	github.com/pressly/goose/v3 v3.27.0
	github.com/prometheus/client_golang v1.23.2
	github.com/robfig/cron/v3 v3.0.1
	github.com/samber/lo v1.53.0
	github.com/saturn4er/boilerplate-go v1.5.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	go.uber.org/fx v1.24.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.1
	golang.org/x/crypto v0.49.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/time v0.15.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.31.1
	k8s.io/api v0.35.3
	k8s.io/apimachinery v0.35.3
	k8s.io/client-go v0.35.3
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/GoAdminGroup/html v0.0.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.13.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.9.1 // indirect
	github.com/go-jose/go-jose/v4 v4.1.4 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.22.5 // indirect
	github.com/go-openapi/jsonreference v0.21.5 // indirect
	github.com/go-openapi/swag v0.25.5 // indirect
	github.com/go-openapi/swag/cmdutils v0.25.5 // indirect
	github.com/go-openapi/swag/conv v0.25.5 // indirect
	github.com/go-openapi/swag/fileutils v0.25.5 // indirect
	github.com/go-openapi/swag/jsonname v0.25.5 // indirect
	github.com/go-openapi/swag/jsonutils v0.25.5 // indirect
	github.com/go-openapi/swag/loading v0.25.5 // indirect
	github.com/go-openapi/swag/mangling v0.25.5 // indirect
	github.com/go-openapi/swag/netutils v0.25.5 // indirect
	github.com/go-openapi/swag/stringutils v0.25.5 // indirect
	github.com/go-openapi/swag/typeutils v0.25.5 // indirect
	github.com/go-openapi/swag/yamlutils v0.25.5 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/gnostic-models v0.7.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/mfridman/interpolate v0.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/morikuni/aec v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.20.1 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/sethvargo/go-retry v0.3.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stretchr/objx v0.5.3 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.67.0 // indirect
	go.opentelemetry.io/otel v1.42.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.42.0 // indirect
	go.opentelemetry.io/otel/metric v1.42.0 // indirect
	go.opentelemetry.io/otel/trace v1.42.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.1 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
	gorm.io/driver/sqlite v1.6.0 // indirect
	gotest.tools/v3 v3.5.2 // indirect
	k8s.io/klog/v2 v2.140.0 // indirect
	k8s.io/kube-openapi v0.0.0-20260330154417-16be699c7b31 // indirect
	k8s.io/utils v0.0.0-20260319190234-28399d86e0b5 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
	xorm.io/builder v0.3.13 // indirect
	xorm.io/xorm v1.3.11 // indirect
)
