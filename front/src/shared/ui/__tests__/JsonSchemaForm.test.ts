import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import JsonSchemaForm from '../JsonSchemaForm.vue'

function mountForm(schema: Record<string, unknown>, modelValue: Record<string, unknown> = {}) {
  return mount(JsonSchemaForm, {
    props: { schema, modelValue },
  })
}

describe('jsonSchemaForm', () => {
  describe('basic field rendering', () => {
    it('renders text input for type: "string"', () => {
      const w = mountForm({
        type: 'object',
        properties: { name: { type: 'string' } },
      })
      expect(w.find('input[type="text"]').exists()).toBe(true)
    })

    it('renders number input for type: "integer"', () => {
      const w = mountForm({
        type: 'object',
        properties: { port: { type: 'integer' } },
      })
      expect(w.find('input[type="number"]').exists()).toBe(true)
    })

    it('renders number input for type: "number"', () => {
      const w = mountForm({
        type: 'object',
        properties: { rate: { type: 'number' } },
      })
      expect(w.find('input[type="number"]').exists()).toBe(true)
    })

    it('renders checkbox for type: "boolean"', () => {
      const w = mountForm({
        type: 'object',
        properties: { enabled: { type: 'boolean' } },
      })
      expect(w.find('input[type="checkbox"]').exists()).toBe(true)
    })

    it('renders select dropdown for type: "string" with enum', () => {
      const w = mountForm({
        type: 'object',
        properties: { mode: { type: 'string', enum: ['fast', 'slow'] } },
      })
      expect(w.find('select').exists()).toBe(true)
      const options = w.findAll('option')
      expect(options.length).toBe(3) // placeholder + 2 options
    })

    it('renders textarea for type: "string" with multiline: true', () => {
      const w = mountForm({
        type: 'object',
        properties: { query: { type: 'string', multiline: true } },
      })
      expect(w.find('textarea').exists()).toBe(true)
    })

    it('renders password input for airbyte_secret: true', () => {
      const w = mountForm({
        type: 'object',
        properties: { api_key: { type: 'string', airbyte_secret: true } },
      })
      expect(w.find('input[type="password"]').exists()).toBe(true)
    })
  })

  describe('labels and metadata', () => {
    it('uses title as label when present', () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string', title: 'Database Host' } },
      })
      expect(w.text()).toContain('Database Host')
    })

    it('falls back to humanized property key when no title', () => {
      const w = mountForm({
        type: 'object',
        properties: { database_name: { type: 'string' } },
      })
      expect(w.text()).toContain('Database name')
    })

    it('shows asterisk for required fields', () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string' } },
        required: ['host'],
      })
      expect(w.find('.text-danger-500').exists()).toBe(true)
      expect(w.find('.text-danger-500').text()).toBe('*')
    })

    it('shows description as help text', () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string', description: 'The hostname of the DB' } },
      })
      expect(w.text()).toContain('The hostname of the DB')
    })

    it('shows examples as placeholder', () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string', examples: ['localhost'] } },
      })
      const input = w.find('input[type="text"]')
      expect(input.attributes('placeholder')).toBe('localhost')
    })
  })

  describe('nested objects', () => {
    it('renders nested object properties as a fieldset', () => {
      const w = mountForm({
        type: 'object',
        properties: {
          tunnel: {
            type: 'object',
            title: 'SSH Tunnel',
            properties: {
              host: { type: 'string' },
              port: { type: 'integer' },
            },
          },
        },
      })
      expect(w.find('fieldset').exists()).toBe(true)
      expect(w.text()).toContain('SSH Tunnel')
      expect(w.findAll('input').length).toBe(2)
    })

    it('supports two levels of nesting', () => {
      const w = mountForm({
        type: 'object',
        properties: {
          outer: {
            type: 'object',
            properties: {
              inner: {
                type: 'object',
                properties: {
                  value: { type: 'string' },
                },
              },
            },
          },
        },
      })
      expect(w.findAll('fieldset').length).toBe(2)
      expect(w.find('input[type="text"]').exists()).toBe(true)
    })
  })

  describe('oneOf conditional rendering', () => {
    const oneOfSchema = {
      type: 'object',
      properties: {
        auth: {
          title: 'Authentication',
          oneOf: [
            {
              title: 'API Key',
              properties: {
                method: { const: 'api_key' },
                api_key: { type: 'string', title: 'API Key' },
              },
              required: ['api_key'],
            },
            {
              title: 'OAuth',
              properties: {
                method: { const: 'oauth' },
                client_id: { type: 'string', title: 'Client ID' },
              },
              required: ['client_id'],
            },
          ],
        },
      },
    }

    it('renders selector for oneOf options', () => {
      const w = mountForm(oneOfSchema)
      const select = w.find('select')
      expect(select.exists()).toBe(true)
      const options = w.findAll('select option')
      expect(options.length).toBe(2)
      expect(options[0].text()).toBe('API Key')
      expect(options[1].text()).toBe('OAuth')
    })

    it('shows only fields from selected oneOf branch', () => {
      const w = mountForm(oneOfSchema, { auth: { method: 'oauth' } })
      expect(w.text()).toContain('Client ID')
    })

    it('auto-sets const discriminator values on branch switch', async () => {
      const w = mountForm(oneOfSchema)
      const select = w.find('select')

      // Trigger change event with branch index 1 (OAuth)
      const changeEvent = new Event('change')
      Object.defineProperty(changeEvent, 'target', { value: { value: '1' } })
      select.element.dispatchEvent(changeEvent)
      await w.vm.$nextTick()

      const emitted = w.emitted('update:modelValue')!
      const oauthEmit = emitted.find(
        e => (e[0] as any).auth?.method === 'oauth',
      )
      expect(oauthEmit).toBeTruthy()
      expect((oauthEmit![0] as any).auth).toEqual({ method: 'oauth' })
    })
  })

  describe('v-model binding', () => {
    it('emits update:modelValue when field value changes', async () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string' } },
      })
      const input = w.find('input[type="text"]')
      await input.setValue('myhost')
      const emitted = w.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      expect((emitted!.at(-1)![0] as any).host).toBe('myhost')
    })

    it('pre-populates fields from modelValue prop', () => {
      const w = mountForm(
        { type: 'object', properties: { host: { type: 'string' } } },
        { host: 'preset-value' },
      )
      const input = w.find('input[type="text"]')
      expect((input.element as HTMLInputElement).value).toBe('preset-value')
    })

    it('handles default values from schema on mount', () => {
      const w = mountForm({
        type: 'object',
        properties: { port: { type: 'integer', default: 5432 } },
      })
      const emitted = w.emitted('update:modelValue')
      expect(emitted).toBeTruthy()
      expect((emitted![0][0] as any).port).toBe(5432)
    })
  })

  describe('validation', () => {
    it('shows error for empty required field on blur', async () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string' } },
        required: ['host'],
      })
      const input = w.find('input[type="text"]')
      await input.trigger('blur')
      expect(w.text()).toContain('required')
    })

    it('clears error when field is corrected', async () => {
      const w = mountForm({
        type: 'object',
        properties: { host: { type: 'string' } },
        required: ['host'],
      })
      const input = w.find('input[type="text"]')
      await input.trigger('blur')
      expect(w.text()).toContain('required')

      // Simulate correction by setting value and re-rendering
      await w.setProps({ modelValue: { host: 'filled' } })
      expect(w.text()).not.toContain('required')
    })
  })
})
