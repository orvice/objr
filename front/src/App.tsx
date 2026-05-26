import { FormEvent, useEffect, useMemo, useRef, useState } from "react"
import { AlertCircle, CheckCircle2, Copy, ExternalLink, ImageIcon, Loader2, UploadCloud } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { cn } from "@/lib/utils"

type UploadResponse = {
  message: string
  url?: string
  content_type?: string
  file_mine?: Record<string, string[]>
}

const defaultApiBaseUrl = import.meta.env.VITE_API_BASE_URL || window.location.origin

function normalizeBaseUrl(value: string) {
  return value.trim().replace(/\/+$/, "")
}

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB"]
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / Math.pow(1024, index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

export default function App() {
  const fileInputRef = useRef<HTMLInputElement | null>(null)
  const [apiBaseUrl, setApiBaseUrl] = useState(defaultApiBaseUrl)
  const [token, setToken] = useState("")
  const [file, setFile] = useState<File | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [error, setError] = useState("")
  const [result, setResult] = useState<UploadResponse | null>(null)
  const [copied, setCopied] = useState(false)

  const previewUrl = useMemo(() => (file ? URL.createObjectURL(file) : ""), [file])

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl)
    }
  }, [previewUrl])

  const selectFile = (nextFile?: File) => {
    setError("")
    setResult(null)
    setCopied(false)

    if (!nextFile) {
      setFile(null)
      return
    }

    if (!nextFile.type.startsWith("image/")) {
      setError("请选择图片文件。")
      return
    }

    setFile(nextFile)
  }

  const upload = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError("")
    setResult(null)
    setCopied(false)

    if (!file) {
      setError("请先选择一张图片。")
      return
    }
    if (!token.trim()) {
      setError("请输入 Token。")
      return
    }

    const base = normalizeBaseUrl(apiBaseUrl)
    if (!base) {
      setError("请输入 API Base URL。")
      return
    }

    const form = new FormData()
    form.append("image", file)

    setIsUploading(true)
    try {
      const response = await fetch(`${base}/v1/image`, {
        method: "POST",
        headers: {
          Token: token.trim(),
        },
        body: form,
      })

      const text = await response.text()
      let payload: UploadResponse
      try {
        payload = text ? JSON.parse(text) : { message: response.statusText }
      } catch {
        payload = { message: text || response.statusText }
      }

      if (!response.ok) {
        throw new Error(payload.message || `上传失败：HTTP ${response.status}`)
      }

      setResult(payload)
    } catch (err) {
      setError(err instanceof Error ? err.message : "上传失败，请稍后重试。")
    } finally {
      setIsUploading(false)
    }
  }

  const copyUrl = async () => {
    if (!result?.url) return
    await navigator.clipboard.writeText(result.url)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }

  return (
    <main className="min-h-screen bg-[radial-gradient(circle_at_top_left,_hsl(var(--primary)/0.18),_transparent_32rem),linear-gradient(180deg,_hsl(var(--background)),_hsl(var(--muted)))] px-4 py-8 text-foreground sm:py-12">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-8">
        <header className="space-y-4 text-center">
          <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/20">
            <ImageIcon className="size-7" />
          </div>
          <div className="space-y-2">
            <p className="text-sm font-medium uppercase tracking-[0.3em] text-muted-foreground">objr image hosting</p>
            <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">简单图床上传</h1>
            <p className="mx-auto max-w-2xl text-muted-foreground">
              选择图片，调用后端 <code className="rounded bg-muted px-1.5 py-0.5">POST /v1/image</code>，快速获取 CDN 链接。
            </p>
          </div>
        </header>

        <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
          <Card className="backdrop-blur">
            <CardHeader>
              <CardTitle>上传图片</CardTitle>
              <CardDescription>请求会携带 Token header，并使用 multipart/form-data 的 image 字段。</CardDescription>
            </CardHeader>
            <CardContent>
              <form className="space-y-5" onSubmit={upload}>
                <div className="grid gap-2">
                  <Label htmlFor="apiBaseUrl">API Base URL</Label>
                  <Input
                    id="apiBaseUrl"
                    placeholder="https://objr.example.com"
                    value={apiBaseUrl}
                    onChange={(event) => setApiBaseUrl(event.target.value)}
                  />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="token">Token</Label>
                  <Input
                    id="token"
                    type="password"
                    autoComplete="off"
                    placeholder="请输入服务端 auth_token"
                    value={token}
                    onChange={(event) => setToken(event.target.value)}
                  />
                </div>

                <div className="grid gap-2">
                  <Label>图片</Label>
                  <button
                    type="button"
                    className={cn(
                      "flex min-h-52 w-full flex-col items-center justify-center rounded-xl border border-dashed border-muted-foreground/30 bg-muted/40 p-6 text-center transition hover:border-primary/60 hover:bg-primary/5",
                      isDragging && "border-primary bg-primary/10",
                    )}
                    onClick={() => fileInputRef.current?.click()}
                    onDragOver={(event) => {
                      event.preventDefault()
                      setIsDragging(true)
                    }}
                    onDragLeave={() => setIsDragging(false)}
                    onDrop={(event) => {
                      event.preventDefault()
                      setIsDragging(false)
                      selectFile(event.dataTransfer.files?.[0])
                    }}
                  >
                    {file && previewUrl ? (
                      <div className="flex flex-col items-center gap-4">
                        <img src={previewUrl} alt="待上传预览" className="max-h-40 rounded-lg border object-contain shadow-sm" />
                        <div>
                          <p className="font-medium">{file.name}</p>
                          <p className="text-sm text-muted-foreground">{file.type || "unknown"} · {formatBytes(file.size)}</p>
                        </div>
                      </div>
                    ) : (
                      <div className="flex flex-col items-center gap-3">
                        <UploadCloud className="size-10 text-muted-foreground" />
                        <div>
                          <p className="font-medium">点击选择或拖拽图片到这里</p>
                          <p className="text-sm text-muted-foreground">支持浏览器可识别的 image/* 文件</p>
                        </div>
                      </div>
                    )}
                  </button>
                  <Input
                    ref={fileInputRef}
                    type="file"
                    accept="image/*"
                    className="hidden"
                    onChange={(event) => selectFile(event.target.files?.[0])}
                  />
                </div>

                {error ? (
                  <Alert variant="destructive">
                    <AlertCircle className="size-4" />
                    <AlertTitle>上传失败</AlertTitle>
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                ) : null}

                <Button type="submit" className="w-full" size="lg" disabled={isUploading}>
                  {isUploading ? <Loader2 className="animate-spin" /> : <UploadCloud />}
                  {isUploading ? "上传中..." : "上传并生成链接"}
                </Button>
              </form>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>上传结果</CardTitle>
              <CardDescription>成功后可复制 CDN URL 或直接打开。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              {result?.url ? (
                <>
                  <Alert>
                    <CheckCircle2 className="size-4 text-emerald-600" />
                    <AlertTitle>上传成功</AlertTitle>
                    <AlertDescription>{result.message || "success"}</AlertDescription>
                  </Alert>

                  <div className="overflow-hidden rounded-xl border bg-muted/30">
                    <img src={result.url} alt="已上传图片" className="max-h-72 w-full object-contain" />
                  </div>

                  <div className="space-y-2">
                    <Label>CDN URL</Label>
                    <div className="flex gap-2">
                      <Input value={result.url} readOnly className="font-mono text-xs" />
                      <Button type="button" variant="outline" size="icon" onClick={copyUrl} title="复制 URL">
                        <Copy />
                      </Button>
                      <Button type="button" variant="outline" size="icon" asChild title="打开 URL">
                        <a href={result.url} target="_blank" rel="noreferrer">
                          <ExternalLink />
                        </a>
                      </Button>
                    </div>
                    {copied ? <p className="text-sm text-emerald-600">已复制到剪贴板</p> : null}
                  </div>

                  <Separator />

                  <dl className="grid gap-3 text-sm">
                    <div className="flex items-center justify-between gap-4">
                      <dt className="text-muted-foreground">Content-Type</dt>
                      <dd className="font-mono">{result.content_type || "-"}</dd>
                    </div>
                  </dl>
                </>
              ) : (
                <div className="flex min-h-[28rem] flex-col items-center justify-center rounded-xl border border-dashed bg-muted/30 p-8 text-center">
                  <ImageIcon className="mb-4 size-12 text-muted-foreground" />
                  <p className="font-medium">还没有上传结果</p>
                  <p className="mt-1 text-sm text-muted-foreground">选择图片并上传后，生成的链接会显示在这里。</p>
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
