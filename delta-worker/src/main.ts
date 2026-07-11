import { consumeReleaseEvents, closeQueue, type ReleasePublishedEvent } from "./queue.ts"
import { generateBsdiff, generateXdelta } from "./delta.ts"
import { readFile } from "node:fs/promises"

const API_URL = (process.env.API_URL ?? "http://localhost:8000").replace(/\/$/, "")
const WORKER_TOKEN = process.env.WORKER_TOKEN ?? ""

function headers(): Record<string, string> {
  const h: Record<string, string> = {}
  if (WORKER_TOKEN) h["X-Worker-Token"] = WORKER_TOKEN
  return h
}

async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { ...headers(), ...(init?.headers as Record<string, string> | undefined) },
  })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`API ${res.status} ${path}: ${body.slice(0, 500)}`)
  }
  return res
}

type PreviousRelease = { id: string; version: string; buildNumber?: string; platform: string }

async function getPreviousReleases(releaseId: string): Promise<PreviousRelease[]> {
  const res = await apiFetch(`/api/v1/worker/releases/${releaseId}/previous`)
  return res.json()
}

async function downloadArtifact(url: string, dest: string): Promise<void> {
  const res = await fetch(url, { headers: headers() })
  if (!res.ok) throw new Error(`download ${res.status}: ${url}`)
  await Bun.write(dest, res)
}

async function uploadDelta(
  releaseId: string,
  deltaPath: string,
  meta: {
    fromVersion: string
    deltaFormat: string
    filename: string
  },
) {
  const form = new FormData()
  form.append("metadata", JSON.stringify(meta))
  form.append("file", Bun.file(deltaPath))
  const res = await fetch(
    `${API_URL}/api/v1/worker/releases/${releaseId}/delta-artifacts`,
    { method: "POST", body: form, headers: headers() },
  )
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`upload ${res.status}: ${body.slice(0, 500)}`)
  }
  return res.json()
}

async function handleEvent(event: ReleasePublishedEvent) {
  const { releaseId, version } = event.data
  console.log(`[delta-worker] release published: ${version} (${releaseId})`)

  const previous = await getPreviousReleases(releaseId)
  console.log(`[delta-worker] ${previous.length} candidate(s) for delta`)

  // Delta generation needs old + new full-artifact URLs.
  // The worker downloads them from the backend's /download endpoint,
  // generates deltas with bsdiff/xdelta3, and uploads them back.
  //
  // Provide old/new artifact URLs via env vars or resolve them from the API:
  //   PREVIOUS_ARTIFACT_URL  — e.g. http://localhost:8000/api/v1/updates/artifacts/<id>/download
  //   NEW_ARTIFACT_URL       — same for the new release
  //   PREVIOUS_VERSION       — version string of the old release

  const oldUrl = process.env.PREVIOUS_ARTIFACT_URL
  const newUrl = process.env.NEW_ARTIFACT_URL
  const fromVersion = process.env.PREVIOUS_VERSION

  if (!oldUrl || !newUrl || !fromVersion) {
    console.log("[delta-worker] skipping — set PREVIOUS_ARTIFACT_URL, NEW_ARTIFACT_URL, PREVIOUS_VERSION")
    return
  }

  const oldFile = `/tmp/clave-delta-old-${releaseId}`
  const newFile = `/tmp/clave-delta-new-${releaseId}`

  try {
    await Promise.all([downloadArtifact(oldUrl, oldFile), downloadArtifact(newUrl, newFile)])

    // Try bsdiff first (apt install bsdiff), fall back to xdelta3
    const strategies = [
      { name: "bsdiff", fn: generateBsdiff, fmt: "bsdiff" },
      { name: "xdelta3", fn: generateXdelta, fmt: "xdelta3" },
    ]

    for (const { name, fn, fmt } of strategies) {
      try {
        const result = await fn(oldFile, newFile)
        console.log(`[delta-worker] ${name} delta: ${result.sizeBytes} bytes`)
        await uploadDelta(releaseId, result.path, {
          fromVersion,
          deltaFormat: fmt,
          filename: `${fromVersion}-${version}.delta`,
        })
        console.log(`[delta-worker] uploaded ${fmt} delta for ${fromVersion} -> ${version}`)
        break
      } catch (err) {
        console.warn(`[delta-worker] ${name} failed:`, (err as Error).message)
      }
    }
  } catch (error) {
    console.error("[delta-worker] failed:", (error as Error).message)
    throw error
  } finally {
    try { await Bun.file(oldFile).exists().then((e) => e && Bun.file(oldFile).delete()) } catch {}
    try { await Bun.file(newFile).exists().then((e) => e && Bun.file(newFile).delete()) } catch {}
  }
}

async function main() {
  console.log("[delta-worker] starting")
  await consumeReleaseEvents(async (event) => {
    try { await handleEvent(event) } catch (error) {
      console.error("[delta-worker] event failed:", (error as Error).message)
      throw error
    }
  })
  console.log("[delta-worker] waiting for events...")
}

async function shutdown(signal: string) {
  console.log(`[delta-worker] shutting down (${signal})`)
  try { await closeQueue() } catch {}
  process.exit(0)
}

process.on("SIGINT", () => shutdown("SIGINT"))
process.on("SIGTERM", () => shutdown("SIGTERM"))

main().catch((error) => {
  console.error("[delta-worker] fatal:", (error as Error).message)
  process.exit(1)
})
