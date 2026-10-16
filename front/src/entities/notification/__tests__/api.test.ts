import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createChannel, listChannels } from '../api'

const notificationClient = vi.hoisted(() => ({
  listNotificationChannels: vi.fn(),
  createNotificationChannel: vi.fn(),
}))

vi.mock('@shared/api/services', () => ({ notificationClient }))

describe('notification api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    ['valid JSON object', '{"webhook_url":"https://x"}', { webhook_url: 'https://x' }, false],
    ['empty config', '', {}, false],
    ['malformed JSON', '{', {}, true],
    ['non-object JSON', '[1]', {}, true],
  ])('parses %s', async (_, config, expected, invalid) => {
    notificationClient.listNotificationChannels.mockResolvedValue({ channels: [{ id: 'c1', config }] })
    const [channel] = await listChannels()
    expect(channel.config).toEqual(expected)
    expect(channel.configInvalid).toBe(invalid)
  })

  it('fails clearly when the create response has no channel', async () => {
    notificationClient.createNotificationChannel.mockResolvedValue({})
    await expect(createChannel({ name: 'n', channelType: 'slack', config: {}, enabled: true }))
      .rejects
      .toThrow('The server response is missing channel')
  })
})
