'use client';

import { useState } from 'react';
import { Check, Copy, Terminal } from 'lucide-react';
import { cn, copyToClipboard } from '@/lib/utils';
import { toastError, toastSuccess } from '@/lib/toast';

interface CodeBlockProps {
  code: string;
  language?: string;
  title?: string;
  showLineNumbers?: boolean;
  className?: string;
}

export function CodeBlock({
  code,
  language = 'bash',
  title,
  showLineNumbers = false,
  className,
}: CodeBlockProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    const success = await copyToClipboard(code);
    if (success) {
      setCopied(true);
      toastSuccess('Copied to clipboard');
      setTimeout(() => setCopied(false), 2000);
    } else {
      toastError('Failed to copy');
    }
  };

  const lines = code.split('\n');

  return (
    <div className={cn('rounded-lg border border-border overflow-hidden', className)}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-muted/50 border-b border-border">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <Terminal className="h-3.5 w-3.5" />
          {title || language}
        </div>
        <button
          onClick={handleCopy}
          className={cn(
            'inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-all',
            copied
              ? 'bg-status-success/10 text-status-success'
              : 'text-muted-foreground hover:text-foreground hover:bg-accent'
          )}
        >
          {copied ? (
            <>
              <Check className="h-3 w-3" />
              Copied
            </>
          ) : (
            <>
              <Copy className="h-3 w-3" />
              Copy
            </>
          )}
        </button>
      </div>

      {/* Code */}
      <div className="overflow-x-auto bg-[#0a0a0f]">
        <pre className="p-4 text-[13px] leading-6 font-mono">
          <code>
            {lines.map((line, i) => (
              <div key={i} className="flex">
                {showLineNumbers && (
                  <span className="select-none text-zinc-600 w-8 text-right mr-4 flex-shrink-0">
                    {i + 1}
                  </span>
                )}
                <span className="text-zinc-300">{line}</span>
              </div>
            ))}
          </code>
        </pre>
      </div>
    </div>
  );
}
