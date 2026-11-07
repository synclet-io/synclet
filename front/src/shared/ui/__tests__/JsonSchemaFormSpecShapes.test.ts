import type { JsonSchema } from '@shared/lib/jsonSchemaUtils'
import type { VueWrapper } from '@vue/test-utils'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import JsonSchemaForm from '../JsonSchemaForm.vue'

type Value = Record<string, unknown>
type Bound = VueWrapper & { model: { value: Value } }

function mountBound(properties: Record<string, unknown>, modelValue: Value = {}, extra: Record<string, unknown> = {}): Bound {
  const model = ref<Value>(modelValue)
  const schema = { type: 'object', properties, ...extra } as JsonSchema
  const Host = defineComponent({
    setup: () => () => h(JsonSchemaForm, {
      'schema': schema,
      'modelValue': model.value,
      'onUpdate:modelValue': (v: Value) => { model.value = v },
    }),
  })
  return Object.assign(mount(Host), { model })
}

function value(w: Bound): Value {
  return w.model.value
}

function validate(w: Bound): Array<{ field: string, message: string }> {
  return (w.findComponent(JsonSchemaForm).vm as any).validate()
}

function messages(w: VueWrapper): string[] {
  return w.findAll('p.text-danger').map(p => p.text())
}

async function selectIndex(w: VueWrapper, selector: string, index: number) {
  const select = w.find(selector)
  ;(select.element as HTMLSelectElement).selectedIndex = index
  await select.trigger('change')
}

// source-gnews: `in`
const gnewsIn = {
  type: 'array',
  title: 'In',
  order: 4,
  items: { type: 'string', enum: ['title', 'description', 'content'] },
}

// source-pinterest: `status`
const pinterestStatus = {
  title: 'Status',
  type: ['array', 'null'],
  items: { type: 'string', enum: ['ACTIVE', 'PAUSED', 'ARCHIVED'] },
  uniqueItems: true,
}

// source-github: `repositories`
const githubRepositories = {
  type: 'array',
  items: { type: 'string', pattern: '^[\\w.-]+/(([\\w.-]*\\*)|[\\w.-]+(?<!\\.git))$' },
  minItems: 1,
  title: 'GitHub Repositories',
  order: 1,
  pattern_descriptor: 'org/repo org/another-repo org/* org/a*',
}

describe('jsonSchemaForm spec shapes', () => {
  describe('arrays', () => {
    it('renders an array of enum as a multi-select that stores the chosen values', async () => {
      const w = mountBound({ in: gnewsIn })
      const boxes = w.findAll('input[type="checkbox"]')
      expect(boxes.map(b => (b.element as HTMLInputElement).value)).toEqual(['title', 'description', 'content'])
      expect(w.find('input[type="text"]').exists()).toBe(false)

      await boxes[2].setValue(true)
      await boxes[0].setValue(true)
      expect(value(w).in).toEqual(['title', 'content'])

      await boxes[2].setValue(false)
      expect(value(w).in).toEqual(['title'])
    })

    it('stores numbers for an array of integers or numbers', async () => {
      const w = mountBound({
        account_ids: { type: 'array', items: { type: 'integer' } },
        ratios: { type: 'array', items: { type: 'number' } },
      })
      const inputs = w.findAll('input[type="text"]')
      await inputs[0].setValue('12, 34')
      await inputs[1].setValue('0.5,1.25')
      expect(value(w).account_ids).toEqual([12, 34])
      expect(value(w).ratios).toEqual([0.5, 1.25])
    })

    it('keeps the comma-separated input for an array without items', async () => {
      const w = mountBound({ tags: { type: 'array', title: 'Tags' } })
      await w.find('input[type="text"]').setValue('a, b')
      expect(value(w).tags).toEqual(['a', 'b'])
    })

    it('validates minItems, maxItems and uniqueItems', async () => {
      const w = mountBound(
        {
          repositories: githubRepositories,
          status: pinterestStatus,
          labels: { type: 'array', items: { type: 'string' }, maxItems: 2 },
        },
        { repositories: [], status: ['ACTIVE', 'ACTIVE'], labels: ['a', 'b', 'c'] },
      )
      const errors = validate(w)
      expect(errors.map((e: any) => e.field).sort()).toEqual(['labels', 'repositories', 'status'])
      await nextTick()
      const text = messages(w).join(' | ')
      expect(text).toContain('at least 1')
      expect(text).toContain('at most 2')
      expect(text).toContain('unique')
    })
  })

  describe('types', () => {
    // source-pinterest: `click_window_days`
    it('stores a non-string enum value with its type', async () => {
      const w = mountBound({
        click_window_days: { title: 'Click window days', default: 30, enum: [0, 1, 7, 14, 30, 60], type: 'integer' },
      })
      expect(value(w).click_window_days).toBe(30)
      await selectIndex(w, 'select', 3)
      expect(value(w).click_window_days).toBe(7)
    })

    it('renders a type list with null as its non-null type', () => {
      const w = mountBound({
        status: pinterestStatus,
        // source-dynamodb: `credentials` branch
        credentials: {
          title: 'Authenticate via Access Keys',
          type: ['null', 'object'],
          properties: { access_key_id: { title: 'Dynamodb Key Id', type: 'string' } },
        },
        region: { title: 'Region', type: ['string', 'null'] },
      })
      expect(w.findAll('input[type="checkbox"]').length).toBe(3)
      expect(w.find('fieldset legend').text()).toContain('Authenticate via Access Keys')
      expect(w.find('#field-access_key_id').attributes('type')).toBe('text')
      expect(w.find('#field-region').attributes('type')).toBe('text')
    })

    // source-declarative: `__manifest`
    it('renders an object without properties as a JSON editor', async () => {
      const w = mountBound({ __manifest: { title: 'Manifest', type: 'object' } })
      const editor = w.find('textarea')
      await editor.setValue('{"version": "6.0.0", "streams": []}')
      expect(value(w).__manifest).toEqual({ version: '6.0.0', streams: [] })

      await editor.setValue('{"version": ')
      expect(w.text()).toContain('Invalid JSON')
      expect(value(w).__manifest).toEqual({ version: '6.0.0', streams: [] })
      expect(validate(w).map((e: any) => e.field)).toEqual(['__manifest'])

      await editor.setValue('[1, 2]')
      expect(w.text()).toContain('JSON object')
    })
  })

  describe('oneOf', () => {
    // destination-gcs: `format.compression_codec` (Avro branch)
    const compressionCodec = {
      title: 'Compression Codec',
      type: 'object',
      oneOf: [
        { title: 'No Compression', required: ['codec'], properties: { codec: { type: 'string', enum: ['no compression'], default: 'no compression' } } },
        {
          title: 'Deflate',
          required: ['codec'],
          properties: {
            codec: { type: 'string', enum: ['Deflate'], default: 'Deflate' },
            compression_level: { title: 'Deflate level', type: 'integer', default: 0, minimum: 0, maximum: 9 },
          },
        },
        { title: 'bzip2', required: ['codec'], properties: { codec: { type: 'string', enum: ['bzip2'], default: 'bzip2' } } },
        {
          title: 'xz',
          required: ['codec'],
          properties: {
            codec: { type: 'string', enum: ['xz'], default: 'xz' },
            compression_level: { title: 'Compression Level', type: 'integer', default: 6, minimum: 0, maximum: 9 },
          },
        },
      ],
    }

    // destination-kafka style: branches told apart only by their properties
    const storage = {
      title: 'Storage',
      type: 'object',
      oneOf: [
        { title: 'Local', properties: { path: { type: 'string', title: 'Path' } } },
        { title: 'Bucket', required: ['bucket'], properties: { bucket: { type: 'string', title: 'Bucket' }, region: { type: 'string', title: 'Region' } } },
      ],
    }

    it('selects a branch without a const discriminator and keeps it selected', async () => {
      const w = mountBound({ compression_codec: compressionCodec })
      await selectIndex(w, 'select', 1)
      expect(value(w).compression_codec).toEqual({ codec: 'Deflate', compression_level: 0 })
      expect((w.find('select').element as HTMLSelectElement).value).toBe('1')
      expect(w.text()).toContain('Deflate level')
    })

    it('keeps a branch without any discriminator selected', async () => {
      const w = mountBound({ storage })
      await selectIndex(w, 'select', 1)
      expect((w.find('select').element as HTMLSelectElement).value).toBe('1')
      expect(w.text()).toContain('Bucket')
      await w.find('#field-region').setValue('eu')
      expect((w.find('select').element as HTMLSelectElement).value).toBe('1')
      expect(value(w).storage).toEqual({ region: 'eu' })
    })

    it('matches a stored value to the branch whose properties fit it', () => {
      const xz = mountBound({ compression_codec: compressionCodec }, { compression_codec: { codec: 'xz', compression_level: 3 } })
      expect((xz.find('select').element as HTMLSelectElement).value).toBe('3')

      const bucket = mountBound({ storage }, { storage: { bucket: 'b1' } })
      expect((bucket.find('select').element as HTMLSelectElement).value).toBe('1')
      expect((bucket.find('#field-bucket').element as HTMLInputElement).value).toBe('b1')
    })

    // destination-clickhouse: `protocol`
    it('stores the const of a primitive branch', async () => {
      const w = mountBound({
        protocol: {
          default: 'native',
          oneOf: [{ const: 'native', title: 'Native' }, { const: 'http', title: 'HTTP' }],
          title: 'Protocol',
          type: 'string',
        },
      })
      expect(value(w).protocol).toBe('native')
      expect((w.find('select').element as HTMLSelectElement).value).toBe('0')
      await selectIndex(w, 'select', 1)
      expect(value(w).protocol).toBe('http')
      expect((w.find('select').element as HTMLSelectElement).value).toBe('1')
    })

    it('renders a primitive branch without a const as an input of its type', async () => {
      const w = mountBound({
        batch_size: {
          title: 'Batch size',
          oneOf: [{ const: 'auto', title: 'Automatic' }, { type: 'integer', title: 'Fixed', minimum: 1 }],
        },
      }, { batch_size: 500 })
      expect((w.find('select').element as HTMLSelectElement).value).toBe('1')
      const input = w.find('input[type="number"]')
      expect((input.element as HTMLInputElement).value).toBe('500')
      await input.setValue('250')
      expect(value(w).batch_size).toBe(250)
    })
  })

  describe('hints and visibility', () => {
    // source-github: `start_date`
    it('renders format date-time as a date-time input that stores the spec format', async () => {
      const w = mountBound({
        start_date: {
          type: 'string',
          title: 'Start date',
          examples: ['2021-03-01T00:00:00Z'],
          pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$',
          pattern_descriptor: 'YYYY-MM-DDTHH:mm:ssZ',
          format: 'date-time',
        },
        end_date: {
          type: 'string',
          format: 'date-time',
          pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{6}Z$',
        },
      }, { end_date: '2024-05-06T07:08:09.000000Z' })
      const start = w.find('#field-start_date')
      expect(start.attributes('type')).toBe('datetime-local')
      await start.setValue('2024-01-02T03:04')
      expect(value(w).start_date).toBe('2024-01-02T03:04:00Z')

      const end = w.find('#field-end_date')
      expect((end.element as HTMLInputElement).value).toBe('2024-05-06T07:08:09')
      await end.setValue('2024-05-06T07:08:10')
      expect(value(w).end_date).toBe('2024-05-06T07:08:10.000000Z')
    })

    it('renders format date as a date input that stores the spec format', async () => {
      const w = mountBound({
        // source-pinterest: `start_date`
        start_date: { type: 'string', format: 'date', pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}$', pattern_descriptor: 'YYYY-MM-DD', examples: ['2022-07-28'] },
        report_date: { type: 'string', format: 'date', pattern: '^[0-9]{2}-[0-9]{2}-[0-9]{4}$' },
      })
      const start = w.find('#field-start_date')
      expect(start.attributes('type')).toBe('date')
      await start.setValue('2022-07-28')
      expect(value(w).start_date).toBe('2022-07-28')

      await w.find('#field-report_date').setValue('2024-01-31')
      expect(value(w).report_date).toBe('01-31-2024')
    })

    // source-salesforce: `start_date`
    const salesforceStartDate = {
      title: 'Start Date',
      type: 'string',
      pattern: '^([0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)?)$',
      pattern_descriptor: 'YYYY-MM-DD or YYYY-MM-DDTHH:mm:ssZ',
      examples: ['2021-07-25', '2021-07-25T00:00:00Z'],
      format: 'date-time',
    }

    // source-mixpanel: `start_date`, `end_date`
    const mixpanelDate = {
      type: 'string',
      format: 'date-time',
      pattern: '^$|^[0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)?$',
      examples: ['2021-11-16'],
    }

    // source-delighted: `since`
    const delightedSince = {
      title: 'Replication Start Date',
      type: 'string',
      examples: ['2022-05-30T04:50:23Z', '2022-05-30 04:50:23'],
      pattern: '^\\d{4}-\\d{2}-\\d{2}[T ]\\d{2}:\\d{2}:\\d{2}Z?$',
      format: 'date-time',
    }

    it('accepts a date on its own for a date-time field', () => {
      const w = mountBound(
        {
          start_date: salesforceStartDate,
          mixpanel_start: mixpanelDate,
          mixpanel_end: mixpanelDate,
          // source-braintree: `start_date`
          braintree_start: { title: 'Start Date', type: 'string', format: 'date-time', examples: ['2020', '2020-12-30', '2020-11-22 20:20:05'] },
        },
        { start_date: '2021-07-25', mixpanel_start: '2021-11-16', mixpanel_end: '2021-11-16T00:00:00Z', braintree_start: '2020-12-30' },
        { required: ['start_date'] },
      )
      expect(validate(w)).toEqual([])
    })

    it('shows a stored value the date input cannot hold as text and leaves it unchanged', async () => {
      const w = mountBound(
        {
          start_date: salesforceStartDate,
          since: delightedSince,
          window_start: { type: 'string', format: 'date-time' },
          precise: { type: 'string', format: 'date-time', pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}.[0-9]{6}Z$' },
        },
        { start_date: '2021-07-25', since: '2022-05-30 04:50:23', window_start: '2024-01-02T03:04:05+02:00', precise: '2024-05-06T07:08:09.123456Z' },
      )
      const stored = { ...value(w) }
      for (const [key, expected] of Object.entries(stored)) {
        const input = w.find(`#field-${key}`)
        expect(input.attributes('type')).toBe('text')
        expect((input.element as HTMLInputElement).value).toBe(expected)
      }
      expect(validate(w)).toEqual([])

      await w.find('#field-since').setValue('2022-06-01 00:00:00')
      expect(value(w)).toEqual({ ...stored, since: '2022-06-01 00:00:00' })
    })

    it('validates date and date-time values', async () => {
      const w = mountBound(
        {
          start_date: { type: 'string', format: 'date', pattern: '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' },
          updated_after: { type: 'string', format: 'date-time' },
        },
        { start_date: '2024-13-40', updated_after: 'yesterday' },
      )
      const errors = validate(w)
      expect(errors.map((e: any) => e.field).sort()).toEqual(['start_date', 'updated_after'])
      await nextTick()
      expect(messages(w).join(' ')).toContain('valid date')
    })

    it('shows pattern_descriptor when a pattern fails', async () => {
      const w = mountBound({
        repository: {
          type: 'string',
          title: 'GitHub Repositories',
          pattern: '^([\\w.-]+/(\\*|[\\w.-]+(?<!\\.git))\\s+)*[\\w.-]+/(\\*|[\\w.-]+(?<!\\.git))$',
          pattern_descriptor: 'org/repo org/another-repo org/*',
        },
        repositories: githubRepositories,
      }, { repositories: ['airbytehq/airbyte.git'] })
      const input = w.find('#field-repository')
      await input.setValue('not a repo')
      await input.trigger('blur')
      expect(w.text()).toContain('org/repo org/another-repo org/*')
      expect(w.text()).not.toContain('(?<!')

      validate(w)
      await nextTick()
      expect(messages(w).join(' | ')).toContain('org/repo org/another-repo org/* org/a*')
    })

    // source-github: `branch`
    it('does not render airbyte_hidden fields but keeps their default', () => {
      const w = mountBound({
        branch: { type: 'string', title: 'Branch', airbyte_hidden: true, default: 'main' },
        token: { type: 'string', title: 'Token' },
      })
      expect(w.text()).not.toContain('Branch')
      expect(w.findAll('input').length).toBe(1)
      expect(value(w).branch).toBe('main')
    })

    it('renders markdown links in descriptions as sanitized links', () => {
      const w = mountBound({
        api_key: {
          type: 'string',
          description: 'Get it from the [API settings](https://example.com/settings). Not [this](javascript:alert(1)).<img src=x onerror=alert(1)>',
        },
      })
      const links = w.findAll('a')
      expect(links[0].attributes('href')).toBe('https://example.com/settings')
      expect(links[0].text()).toBe('API settings')
      expect(links[0].attributes('target')).toBe('_blank')
      expect(w.html()).not.toContain('javascript:')
      expect(w.html()).not.toContain('onerror')
    })
  })

  describe('validation of the whole value', () => {
    const schema = {
      host: { type: 'string', title: 'Host' },
      tunnel: {
        type: 'object',
        title: 'SSH Tunnel',
        required: ['tunnel_host'],
        properties: { tunnel_host: { type: 'string', title: 'Tunnel host' } },
      },
      auth: {
        title: 'Authentication',
        oneOf: [
          { title: 'API Key', required: ['api_key'], properties: { method: { const: 'api_key' }, api_key: { type: 'string' } } },
          { title: 'OAuth', required: ['client_id'], properties: { method: { const: 'oauth' }, client_id: { type: 'string' } } },
        ],
      },
      ports: { type: 'array', items: { type: 'integer', minimum: 1 } },
    }

    it('reports nested objects, the selected oneOf branch and array items, and marks each field', async () => {
      const w = mountBound(schema, { tunnel: {}, auth: { method: 'oauth' }, ports: [22, 0] }, { required: ['host', 'tunnel', 'auth'] })
      expect(messages(w)).toEqual([])

      const errors = validate(w)
      expect(errors.map((e: any) => e.field).sort()).toEqual(['auth.client_id', 'host', 'ports', 'tunnel.tunnel_host'])
      await nextTick()
      expect(messages(w).length).toBe(4)
      expect(w.find('#field-client_id').classes()).toContain('border-danger')
      expect(w.find('#field-tunnel_host').classes()).toContain('border-danger')

      await w.find('#field-host').setValue('db')
      await w.find('#field-client_id').setValue('id')
      await w.find('#field-tunnel_host').setValue('bastion')
      await w.findAll('input[type="text"]').find(i => (i.element as HTMLInputElement).value.includes('22'))!.setValue('22, 23')
      expect(validate(w)).toEqual([])
      await nextTick()
      expect(messages(w)).toEqual([])
    })

    // source-paystack: `secret_key`; the edit page receives stored secrets masked
    const paystackSecret = { type: 'string', title: 'Secret Key', pattern: '^(s|r)k_(live|test)_[a-zA-Z0-9]+$', airbyte_secret: true, minLength: 12 }

    it('accepts the secret mask of a stored config wherever the secret sits', () => {
      const w = mountBound(
        {
          secret_key: paystackSecret,
          tunnel: { type: 'object', required: ['tunnel_key'], properties: { tunnel_key: paystackSecret } },
          credentials: {
            oneOf: [
              { title: 'OAuth', required: ['refresh_token'], properties: { auth_type: { const: 'oauth' }, refresh_token: paystackSecret } },
              { title: 'Key', required: ['api_key'], properties: { auth_type: { const: 'key' }, api_key: paystackSecret } },
            ],
          },
          extra_keys: { type: 'array', items: paystackSecret },
        },
        { secret_key: '********', tunnel: { tunnel_key: '********' }, credentials: { auth_type: 'oauth', refresh_token: '********' }, extra_keys: ['********'] },
        { required: ['secret_key'] },
      )
      expect(validate(w)).toEqual([])
    })

    it('still checks a secret the user typed', () => {
      const w = mountBound({ secret_key: paystackSecret }, { secret_key: 'not-a-key' })
      expect(validate(w).map(e => e.field)).toEqual(['secret_key'])
    })

    it('validates a required oneOf branch that has no value yet', () => {
      const w = mountBound(schema, { host: 'db' }, { required: ['auth'] })
      expect(validate(w).map((e: any) => e.field)).toEqual(['auth.api_key'])
    })
  })
})
