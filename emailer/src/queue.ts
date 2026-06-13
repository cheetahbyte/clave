import { connect } from "amqp-connection-manager"
import type { ConfirmChannel, ConsumeMessage } from "amqplib"
import type { EmailEvent } from "./templates.ts"
import { errMessage } from "./log.ts"

const RABBITMQ_URL = process.env.RABBITMQ_URL ?? "amqp://localhost"

const EXCHANGE = "clave.events"
const QUEUE = "clave.email-worker"
// "#" = all routing keys on the exchange; worker routes by event.type
const ROUTING_KEY = "#"

const connection = connect([RABBITMQ_URL], {
  heartbeatIntervalInSeconds: 5,
  reconnectTimeInSeconds: 5,
})

connection.on("connect", () => {
  console.log("RabbitMQ connected")
})

connection.on("disconnect", ({ err }: { err?: Error }) => {
  console.error("RabbitMQ disconnected", err?.message)
})

export const channel = connection.createChannel({
  json: true,

  setup: async (ch: ConfirmChannel) => {
    await ch.assertExchange(EXCHANGE, "topic", { durable: true })

    await ch.assertQueue(QUEUE, {
      durable: true,
      deadLetterExchange: `${EXCHANGE}.dlx`,
    })

    await ch.assertExchange(`${EXCHANGE}.dlx`, "topic", { durable: true })
    await ch.bindQueue(QUEUE, EXCHANGE, ROUTING_KEY)

    await ch.prefetch(10)
  },
})

export async function publishEmailEvent(event: EmailEvent) {
  await channel.publish(EXCHANGE, event.type, event, {
    persistent: true,
    contentType: "application/json",
  })
}

export async function consumeEmailEvents(
  handler: (event: EmailEvent) => Promise<void>,
) {
  await channel.addSetup(async (ch: ConfirmChannel) => {
    await ch.consume(QUEUE, async (msg: ConsumeMessage | null) => {
      if (!msg) return

      try {
        const event = JSON.parse(msg.content.toString()) as EmailEvent

        await handler(event)

        ch.ack(msg)
      } catch (error) {
        // Log scrubbed message only — raw error/payload may contain PII
        console.error("Failed to process message:", errMessage(error))

        // false, false = reject and do not requeue (goes to DLX)
        ch.nack(msg, false, false)
      }
    })
  })
}

export async function closeQueue() {
  await channel.close()
  await connection.close()
}
