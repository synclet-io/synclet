{{/*
Chart name truncated to 63 chars.
*/}}
{{- define "synclet.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Fully qualified app name.
*/}}
{{- define "synclet.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart label value.
*/}}
{{- define "synclet.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels (no component).
*/}}
{{- define "synclet.labels" -}}
helm.sh/chart: {{ include "synclet.chart" . }}
app.kubernetes.io/name: {{ include "synclet.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "synclet.name" . }}
{{- end }}

{{/*
Base selector labels (no component).
*/}}
{{- define "synclet.selectorLabels" -}}
app.kubernetes.io/name: {{ include "synclet.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/* ---- Server ---- */}}
{{- define "synclet.server.labels" -}}
{{ include "synclet.labels" . }}
app.kubernetes.io/component: server
{{- end }}

{{- define "synclet.server.selectorLabels" -}}
{{ include "synclet.selectorLabels" . }}
app.kubernetes.io/component: server
{{- end }}

{{/* ---- Jobs ---- */}}
{{- define "synclet.jobs.labels" -}}
{{ include "synclet.labels" . }}
app.kubernetes.io/component: jobs
{{- end }}

{{- define "synclet.jobs.selectorLabels" -}}
{{ include "synclet.selectorLabels" . }}
app.kubernetes.io/component: jobs
{{- end }}

{{/* ---- Executor ---- */}}
{{- define "synclet.executor.labels" -}}
{{ include "synclet.labels" . }}
app.kubernetes.io/component: executor
{{- end }}

{{- define "synclet.executor.selectorLabels" -}}
{{ include "synclet.selectorLabels" . }}
app.kubernetes.io/component: executor
{{- end }}

{{/* ---- PostgreSQL ---- */}}
{{- define "synclet.postgres.labels" -}}
{{ include "synclet.labels" . }}
app.kubernetes.io/component: postgres
{{- end }}

{{- define "synclet.postgres.selectorLabels" -}}
{{ include "synclet.selectorLabels" . }}
app.kubernetes.io/component: postgres
{{- end }}

{{/*
ServiceAccount name.
*/}}
{{- define "synclet.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "synclet.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Whether the executor deployment runs under an account of its own.
*/}}
{{- define "synclet.executor.ownServiceAccount" -}}
{{- if and (eq .Values.mode "distributed") .Values.executor.enabled (or .Values.executor.serviceAccount.create .Values.executor.serviceAccount.name) }}true{{- end }}
{{- end }}

{{/*
ServiceAccount name of the executor deployment: its own account when one is
created or named, else the chart's.
*/}}
{{- define "synclet.executor.serviceAccountName" -}}
{{- if .Values.executor.serviceAccount.create }}
{{- default (printf "%s-executor" (include "synclet.fullname" .)) .Values.executor.serviceAccount.name }}
{{- else if .Values.executor.serviceAccount.name }}
{{- .Values.executor.serviceAccount.name }}
{{- else }}
{{- include "synclet.serviceAccountName" . }}
{{- end }}
{{- end }}

{{/*
Default ServiceAccount name of connector pods; empty means the k8s.namespace
default account.
*/}}
{{- define "synclet.pipeline.serviceAccountName" -}}
{{- if .Values.pipeline.serviceAccount.create }}
{{- default (printf "%s-connector" (include "synclet.fullname" .)) .Values.pipeline.serviceAccount.name }}
{{- else }}
{{- .Values.pipeline.serviceAccount.name | default .Values.pipeline.resources.serviceAccountName }}
{{- end }}
{{- end }}

{{/*
Full container image reference.
*/}}
{{- define "synclet.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.image.registry -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .Values.image.repository $tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
{{- end -}}

{{/*
PostgreSQL container image reference.
*/}}
{{- define "synclet.postgresql.image" -}}
{{- $registry := .Values.global.imageRegistry | default .Values.postgresql.image.registry -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .Values.postgresql.image.repository .Values.postgresql.image.tag -}}
{{- else -}}
{{- printf "%s:%s" .Values.postgresql.image.repository .Values.postgresql.image.tag -}}
{{- end -}}
{{- end -}}

{{/*
Chart-managed Secret name.
*/}}
{{- define "synclet.secretName" -}}
{{- printf "%s-secrets" (include "synclet.fullname" .) -}}
{{- end -}}

{{/*
Database DSN — built-in PostgreSQL or external database.
Passwords are URL-encoded to handle special characters safely.
*/}}
{{- define "synclet.databaseDSN" -}}
{{- if .Values.postgresql.enabled -}}
{{- printf "postgres://%s:%s@%s-postgres:5432/%s?sslmode=disable" .Values.postgresql.auth.username (include "synclet.postgresPassword" . | urlquery) (include "synclet.fullname" .) .Values.postgresql.auth.database -}}
{{- else -}}
{{- printf "postgres://%s:%s@%s:%v/%s?sslmode=%s" .Values.externalDatabase.user (.Values.externalDatabase.password | urlquery) .Values.externalDatabase.host (.Values.externalDatabase.port | default 5432) .Values.externalDatabase.database .Values.externalDatabase.sslmode -}}
{{- end -}}
{{- end -}}

{{/*
Internal server address for K8s connector pods to call back.
*/}}
{{- define "synclet.internalServerAddr" -}}
{{- printf "http://%s-internal.%s.svc.cluster.local:%d" (include "synclet.fullname" .) .Release.Namespace (int .Values.server.containerPorts.internal) -}}
{{- end -}}

{{/* ========== Auto-generated secrets (persist across upgrades via lookup) ========== */}}

{{- define "synclet.jwtSecret" -}}
{{- if .Values.auth.jwtSecret -}}
{{- .Values.auth.jwtSecret -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace (include "synclet.secretName" .) -}}
{{- if and $existing $existing.data (index $existing.data "jwt-secret") -}}
{{- index $existing.data "jwt-secret" | b64dec -}}
{{- else -}}
{{- randAlphaNum 32 -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "synclet.encryptionKey" -}}
{{- if .Values.encryption.key -}}
{{- .Values.encryption.key -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace (include "synclet.secretName" .) -}}
{{- if and $existing $existing.data (index $existing.data "encryption-key") -}}
{{- index $existing.data "encryption-key" | b64dec -}}
{{- else -}}
{{- randBytes 32 -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "synclet.internalApiSecret" -}}
{{- if .Values.internalApi.secret -}}
{{- .Values.internalApi.secret -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace (include "synclet.secretName" .) -}}
{{- if and $existing $existing.data (index $existing.data "internal-api-secret") -}}
{{- index $existing.data "internal-api-secret" | b64dec -}}
{{- else -}}
{{- randAlphaNum 32 -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "synclet.postgresPassword" -}}
{{- if .Values.postgresql.auth.password -}}
{{- .Values.postgresql.auth.password -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace (include "synclet.secretName" .) -}}
{{- if and $existing $existing.data (index $existing.data "postgres-password") -}}
{{- index $existing.data "postgres-password" | b64dec -}}
{{- else -}}
{{- randAlphaNum 16 -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Public URL: server.publicUrl, or derived from the only enabled exposure (Ingress or HTTPRoute).
Empty when neither is enabled or the HTTPRoute has no hostnames.
*/}}
{{- define "synclet.publicURL" -}}
{{- $ingress := .Values.server.ingress -}}
{{- $gateway := .Values.server.gateway -}}
{{- if .Values.server.publicUrl -}}
{{- .Values.server.publicUrl | trimSuffix "/" -}}
{{- else if and $ingress.enabled $gateway.enabled -}}
{{- fail "server.ingress.enabled and server.gateway.enabled are both true: set server.publicUrl to the URL users reach Synclet at" -}}
{{- else if $ingress.enabled -}}
{{- printf "%s://%s" (ternary "https" "http" $ingress.tls) $ingress.hostname -}}
{{- else if and $gateway.enabled $gateway.hostnames -}}
{{- printf "%s://%s" (ternary "https" "http" $gateway.tls) (first $gateway.hostnames) -}}
{{- end -}}
{{- end -}}

{{/*
Frontend URL: frontend.url, else the public URL, else the local dev server.
*/}}
{{- define "synclet.frontendURL" -}}
{{- $public := include "synclet.publicURL" . -}}
{{- .Values.frontend.url | default $public | default "http://localhost:5173" -}}
{{- end -}}

{{/*
Comma-separated OIDC provider slugs.
*/}}
{{- define "synclet.oidcProviderSlugs" -}}
{{- $slugs := list -}}
{{- range .Values.oidc.providers -}}
{{- $slugs = append $slugs .slug -}}
{{- end -}}
{{- join "," $slugs -}}
{{- end -}}

{{/*
SMTP password (inline value only; empty = no password).
*/}}
{{/*
"true" when the chart-managed Secret holds the log archive S3 keys, else empty.
*/}}
{{- define "synclet.logArchive.chartManagedS3Keys" -}}
{{- if and (eq .Values.logArchive.backend "s3") .Values.logArchive.s3.accessKeyID (not .Values.logArchive.s3.existingSecret) -}}
true
{{- end -}}
{{- end }}

{{/*
Log archive volume mounts, volumes and env for the server and jobs pods: the
filesystem backend's PVC and the GCS service account key.
*/}}
{{- define "synclet.logArchive.gcsKeyDir" -}}
/var/run/secrets/synclet/gcs
{{- end }}

{{- define "synclet.logArchive.volumeMount" -}}
{{- if and (eq .Values.logArchive.backend "filesystem") .Values.logArchive.persistence.existingClaim }}
- name: log-archive
  mountPath: {{ .Values.logArchive.dir }}
{{- end }}
{{- if and (eq .Values.logArchive.backend "gcs") .Values.logArchive.gcs.credentialsSecret }}
- name: log-archive-gcs-key
  mountPath: {{ include "synclet.logArchive.gcsKeyDir" . }}
  readOnly: true
{{- end }}
{{- end }}

{{- define "synclet.logArchive.volume" -}}
{{- if and (eq .Values.logArchive.backend "filesystem") .Values.logArchive.persistence.existingClaim }}
- name: log-archive
  persistentVolumeClaim:
    claimName: {{ .Values.logArchive.persistence.existingClaim }}
{{- end }}
{{- if and (eq .Values.logArchive.backend "gcs") .Values.logArchive.gcs.credentialsSecret }}
- name: log-archive-gcs-key
  secret:
    secretName: {{ .Values.logArchive.gcs.credentialsSecret }}
    items:
      - key: {{ .Values.logArchive.gcs.credentialsSecretKey }}
        path: key.json
{{- end }}
{{- end }}

{{- define "synclet.logArchive.env" -}}
{{- if and (eq .Values.logArchive.backend "gcs") .Values.logArchive.gcs.credentialsSecret }}
- name: PIPELINE_LOG_ARCHIVE_GCS_CREDENTIALS_FILE
  value: {{ printf "%s/key.json" (include "synclet.logArchive.gcsKeyDir" .) | quote }}
{{- end }}
{{- end }}

{{/*
Log archive backend Synclet runs with: s3 when the bundled Garage is enabled.
*/}}
{{- define "synclet.logArchive.backend" -}}
{{- if .Values.logArchive.garage.enabled -}}
{{- if ne .Values.logArchive.backend "none" -}}
{{- fail (printf "logArchive.garage.enabled selects the s3 backend and wires it to Garage; leave logArchive.backend at none (got %q)" .Values.logArchive.backend) -}}
{{- end -}}
s3
{{- else -}}
{{- .Values.logArchive.backend -}}
{{- end -}}
{{- end }}

{{/* ---- Garage ---- */}}
{{- define "synclet.garage.fullname" -}}
{{- printf "%s-garage" (include "synclet.fullname" .) -}}
{{- end }}

{{- define "synclet.garage.labels" -}}
{{ include "synclet.labels" . }}
app.kubernetes.io/component: garage
{{- end }}

{{- define "synclet.garage.selectorLabels" -}}
{{ include "synclet.selectorLabels" . }}
app.kubernetes.io/component: garage
{{- end }}

{{- define "synclet.garage.image" -}}
{{- $image := .Values.logArchive.garage.image -}}
{{- $registry := .Values.global.imageRegistry | default $image.registry -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry $image.repository $image.tag -}}
{{- else -}}
{{- printf "%s:%s" $image.repository $image.tag -}}
{{- end -}}
{{- end }}

{{- define "synclet.garage.region" -}}
garage
{{- end }}

{{- define "synclet.garage.secretName" -}}
{{- .Values.logArchive.garage.existingSecret | default (include "synclet.garage.fullname" .) -}}
{{- end }}

{{/*
Value of one key in the chart-generated Garage Secret: the stored one when the Secret
exists, otherwise a fresh random value. The access key id follows Garage's own format
(GK + 24 hex chars); the RPC secret and the secret access key are 64 hex chars.
*/}}
{{- define "synclet.garage.secretValue" -}}
{{- $existing := lookup "v1" "Secret" .ctx.Release.Namespace (include "synclet.garage.fullname" .ctx) -}}
{{- if and $existing $existing.data (index $existing.data .key) -}}
{{- index $existing.data .key | b64dec -}}
{{- else if eq .key "access-key-id" -}}
{{- printf "GK%s" (randAlphaNum 64 | sha256sum | trunc 24) -}}
{{- else -}}
{{- randAlphaNum 64 | sha256sum -}}
{{- end -}}
{{- end }}

{{- define "synclet.smtp.secretRef" -}}
{{- if .Values.smtp.existingSecret }}
name: {{ .Values.smtp.existingSecret }}
key: {{ .Values.smtp.existingSecretKey }}
{{- else }}
name: {{ include "synclet.secretName" . }}
key: smtp-password
{{- end }}
{{- end }}

{{/* ========== Secret references (existingSecret or chart-managed) ========== */}}

{{/*
VSO-managed secret name for app secrets.
*/}}
{{- define "synclet.vso.appSecretName" -}}
{{- printf "%s-vault-app" (include "synclet.fullname" .) -}}
{{- end -}}

{{/*
VSO-managed secret name for database.
*/}}
{{- define "synclet.vso.dbSecretName" -}}
{{- printf "%s-vault-db" (include "synclet.fullname" .) -}}
{{- end -}}

{{- define "synclet.auth.secretRef" -}}
{{- if .Values.auth.existingSecret }}
name: {{ .Values.auth.existingSecret }}
key: {{ .Values.auth.existingSecretKey }}
{{- else if .Values.vso.enabled }}
name: {{ include "synclet.vso.appSecretName" . }}
key: jwt-secret
{{- else }}
name: {{ include "synclet.secretName" . }}
key: jwt-secret
{{- end }}
{{- end }}

{{- define "synclet.encryption.secretRef" -}}
{{- if .Values.encryption.existingSecret }}
name: {{ .Values.encryption.existingSecret }}
key: {{ .Values.encryption.existingSecretKeyKey }}
{{- else if .Values.vso.enabled }}
name: {{ include "synclet.vso.appSecretName" . }}
key: encryption-key
{{- else }}
name: {{ include "synclet.secretName" . }}
key: encryption-key
{{- end }}
{{- end }}

{{- define "synclet.internalApi.secretRef" -}}
{{- if .Values.internalApi.existingSecret }}
name: {{ .Values.internalApi.existingSecret }}
key: {{ .Values.internalApi.existingSecretKey }}
{{- else if .Values.vso.enabled }}
name: {{ include "synclet.vso.appSecretName" . }}
key: internal-api-secret
{{- else }}
name: {{ include "synclet.secretName" . }}
key: internal-api-secret
{{- end }}
{{- end }}

{{/*
Secret env vars of the control-plane deployments (server and jobs).
*/}}
{{- define "synclet.secretEnvVars" -}}
- name: AUTH_JWT_SECRET
  valueFrom:
    secretKeyRef:
      {{- include "synclet.auth.secretRef" . | nindent 6 }}
- name: SECRET_ENCRYPTION_KEY
  valueFrom:
    secretKeyRef:
      {{- include "synclet.encryption.secretRef" . | nindent 6 }}
- name: PIPELINE_EXECUTOR_API_TOKEN
  valueFrom:
    secretKeyRef:
      {{- include "synclet.internalApi.secretRef" . | nindent 6 }}
{{- if .Values.encryption.keyPrevious }}
- name: SECRET_ENCRYPTION_KEY_PREVIOUS
  valueFrom:
    secretKeyRef:
      name: {{ include "synclet.secretName" . }}
      key: encryption-key-previous
{{- else if and .Values.encryption.existingSecret .Values.encryption.enableKeyPrevious }}
- name: SECRET_ENCRYPTION_KEY_PREVIOUS
  valueFrom:
    secretKeyRef:
      name: {{ .Values.encryption.existingSecret }}
      key: {{ .Values.encryption.existingSecretKeyPreviousKey }}
{{- else if and .Values.vso.enabled .Values.encryption.enableKeyPrevious }}
- name: SECRET_ENCRYPTION_KEY_PREVIOUS
  valueFrom:
    secretKeyRef:
      name: {{ include "synclet.vso.appSecretName" . }}
      key: encryption-key-previous
{{- end }}
- name: DB_DSN
  valueFrom:
    secretKeyRef:
      {{- if .Values.externalDatabase.existingSecret }}
      name: {{ .Values.externalDatabase.existingSecret }}
      key: {{ .Values.externalDatabase.existingSecretKey }}
      {{- else if and .Values.vso.enabled .Values.vso.database.enabled }}
      name: {{ include "synclet.vso.dbSecretName" . }}
      key: db-dsn
      {{- else }}
      name: {{ include "synclet.secretName" . }}
      key: db-dsn
      {{- end }}
{{- if .Values.smtp.password }}
- name: NOTIFY_SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      {{- include "synclet.smtp.secretRef" . | nindent 6 }}
{{- else if .Values.smtp.existingSecret }}
- name: NOTIFY_SMTP_PASSWORD
  valueFrom:
    secretKeyRef:
      {{- include "synclet.smtp.secretRef" . | nindent 6 }}
{{- end }}
{{- range .Values.oidc.providers }}
- name: AUTH_OIDC_{{ upper .slug }}_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      {{- if $.Values.oidc.existingSecret }}
      name: {{ $.Values.oidc.existingSecret }}
      key: oidc-{{ .slug }}-client-secret
      {{- else }}
      name: {{ include "synclet.secretName" $ }}
      key: oidc-{{ .slug }}-client-secret
      {{- end }}
{{- end }}
{{- if .Values.logArchive.garage.enabled }}
- name: PIPELINE_LOG_ARCHIVE_S3_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      name: {{ include "synclet.garage.secretName" . }}
      key: access-key-id
- name: PIPELINE_LOG_ARCHIVE_S3_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      name: {{ include "synclet.garage.secretName" . }}
      key: secret-access-key
{{- else if eq .Values.logArchive.backend "s3" }}
{{- if or .Values.logArchive.s3.existingSecret .Values.logArchive.s3.accessKeyID }}
- name: PIPELINE_LOG_ARCHIVE_S3_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      {{- if .Values.logArchive.s3.existingSecret }}
      name: {{ .Values.logArchive.s3.existingSecret }}
      key: {{ .Values.logArchive.s3.existingSecretAccessKeyIDKey }}
      {{- else }}
      name: {{ include "synclet.secretName" . }}
      key: log-archive-s3-access-key-id
      {{- end }}
- name: PIPELINE_LOG_ARCHIVE_S3_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      {{- if .Values.logArchive.s3.existingSecret }}
      name: {{ .Values.logArchive.s3.existingSecret }}
      key: {{ .Values.logArchive.s3.existingSecretSecretAccessKeyKey }}
      {{- else }}
      name: {{ include "synclet.secretName" . }}
      key: log-archive-s3-secret-access-key
      {{- end }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Secret env vars of executor pods: only the token they authenticate to the control plane with.
*/}}
{{- define "synclet.executorSecretEnvVars" -}}
- name: EXECUTOR_API_TOKEN
  valueFrom:
    secretKeyRef:
      {{- include "synclet.internalApi.secretRef" . | nindent 6 }}
{{- end }}

{{/*
wait-postgres init container for pods that connect to the bundled PostgreSQL.
Call with (dict "ctx" $ "securityContext" .Values.<component>.containerSecurityContext).
*/}}
{{- define "synclet.waitPostgresInitContainer" -}}
- name: wait-postgres
  image: {{ include "synclet.postgresql.image" .ctx }}
  imagePullPolicy: {{ .ctx.Values.image.pullPolicy }}
  command: ["sh", "-c", "until pg_isready -h {{ include "synclet.fullname" .ctx }}-postgres -p 5432 -U {{ .ctx.Values.postgresql.auth.username }}; do echo 'waiting for postgres...'; sleep 2; done"]
  {{- if .securityContext.enabled }}
  securityContext:
    allowPrivilegeEscalation: {{ .securityContext.allowPrivilegeEscalation }}
    readOnlyRootFilesystem: true
    capabilities:
      drop:
        {{- toYaml .securityContext.capabilities.drop | nindent 8 }}
  {{- end }}
{{- end }}

{{/*
migrate init container: applies pending migrations before the main container
starts. Concurrent runs are safe, each module's migrations hold a Postgres
advisory lock. Call with (dict "ctx" $ "securityContext" .Values.<component>.containerSecurityContext).
*/}}
{{- define "synclet.migrateInitContainer" -}}
- name: migrate
  image: {{ include "synclet.image" .ctx }}
  imagePullPolicy: {{ .ctx.Values.image.pullPolicy }}
  command: ["synclet", "migrate", "up"]
  envFrom:
    - configMapRef:
        name: {{ include "synclet.fullname" .ctx }}
  env:
    {{- include "synclet.secretEnvVars" .ctx | nindent 4 }}
  {{- if .securityContext.enabled }}
  securityContext:
    allowPrivilegeEscalation: {{ .securityContext.allowPrivilegeEscalation }}
    readOnlyRootFilesystem: {{ .securityContext.readOnlyRootFilesystem }}
    capabilities:
      drop:
        {{- toYaml .securityContext.capabilities.drop | nindent 8 }}
  {{- end }}
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
  volumeMounts:
    - name: tmp
      mountPath: /tmp
{{- end }}
