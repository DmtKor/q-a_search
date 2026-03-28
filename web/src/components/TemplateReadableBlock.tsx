import type { ReadableSegment } from '@/api/types'

interface TemplateReadableBlockProps {
  segments: ReadableSegment[]
  className?: string
  style?: React.CSSProperties
}

export function TemplateReadableBlock({ segments, className, style }: TemplateReadableBlockProps) {
  return (
    <div
      className={className}
      style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word', ...style }}
    >
      {segments.map((seg, i) => {
        if (seg.type === 'literal' && seg.text != null) {
          return <span key={i}>{seg.text}</span>
        }
        if (seg.type === 'readable' && seg.code != null) {
          return (
            <strong
              key={i}
              title={seg.code}
              style={{ fontWeight: 700 }}
            >
              {seg.description ?? seg.code}
            </strong>
          )
        }
        if (seg.type === 'raw' && seg.code != null) {
          return (
            <strong key={i} style={{ fontWeight: 700 }}>
              {seg.code}
            </strong>
          )
        }
        return null
      })}
    </div>
  )
}
