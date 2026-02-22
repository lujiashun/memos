import type { Element } from "hast";
import { ChevronDown, ChevronUp } from "lucide-react";
import { memo } from "react";
import toast from "react-hot-toast";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import rehypeRaw from "rehype-raw";
import rehypeSanitize from "rehype-sanitize";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import { remarkDisableSetext } from "@/utils/remark-plugins/remark-disable-setext";
import { remarkPreserveType } from "@/utils/remark-plugins/remark-preserve-type";
import { remarkTag } from "@/utils/remark-plugins/remark-tag";
import { CodeBlock } from "./CodeBlock";
import { isTagNode, isTaskListItemNode } from "./ConditionalComponent";
import { COMPACT_MODE_CONFIG, SANITIZE_SCHEMA } from "./constants";
import { useCompactLabel, useCompactMode } from "./hooks";
import { Blockquote, Heading, HorizontalRule, Image, InlineCode, Link, List, ListItem, Paragraph } from "./markdown";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "./Table";
import { Tag } from "./Tag";
import { TaskListItem } from "./TaskListItem";
import type { MemoContentProps } from "./types";

const MemoContent = (props: MemoContentProps) => {
  const { className, contentClassName, content, onClick, onDoubleClick } = props;
  const t = useTranslate();
  const {
    containerRef: memoContentContainerRef,
    mode: showCompactMode,
    toggle: toggleCompactMode,
  } = useCompactMode(Boolean(props.compact));

  const compactLabel = useCompactLabel(showCompactMode, t as (key: string) => string);

  return (
    <div className={`w-full flex flex-col justify-start items-start text-foreground ${className || ""}`}>
      <div
        ref={memoContentContainerRef}
        className={cn(
          "relative w-full max-w-full wrap-break-word text-base leading-6",
          "[&>*:last-child]:mb-0",
          showCompactMode === "ALL" && "overflow-hidden",
          contentClassName,
        )}
        style={showCompactMode === "ALL" ? { maxHeight: `${COMPACT_MODE_CONFIG.maxHeightVh}vh` } : undefined}
        onMouseUp={onClick}
        onDoubleClick={onDoubleClick}
      >
        <ReactMarkdown
          remarkPlugins={[remarkDisableSetext, remarkMath, remarkGfm, remarkBreaks, remarkTag, remarkPreserveType]}
          rehypePlugins={[rehypeRaw, [rehypeSanitize, SANITIZE_SCHEMA], rehypeKatex]}
          components={{
            // Child components consume from MemoViewContext directly
            input: ((inputProps: React.ComponentProps<"input"> & { node?: Element }) => {
              if (inputProps.node && isTaskListItemNode(inputProps.node)) {
                return <TaskListItem {...inputProps} />;
              }
              return <input {...inputProps} />;
            }) as React.ComponentType<React.ComponentProps<"input">>,
            span: ((spanProps: React.ComponentProps<"span"> & { node?: Element }) => {
              const { node, ...rest } = spanProps;
              if (node && isTagNode(node)) {
                return <Tag {...spanProps} />;
              }
              return <span {...rest} />;
            }) as React.ComponentType<React.ComponentProps<"span">>,
            // Headings
            h1: ({ children }) => <Heading level={1}>{children}</Heading>,
            h2: ({ children }) => <Heading level={2}>{children}</Heading>,
            h3: ({ children }) => <Heading level={3}>{children}</Heading>,
            h4: ({ children }) => <Heading level={4}>{children}</Heading>,
            h5: ({ children }) => <Heading level={5}>{children}</Heading>,
            h6: ({ children }) => <Heading level={6}>{children}</Heading>,
            // Block elements
            p: ({ children }) => <Paragraph>{children}</Paragraph>,
            blockquote: ({ children }) => <Blockquote>{children}</Blockquote>,
            hr: () => <HorizontalRule />,
            // Lists
            ul: ({ children, ...props }) => <List {...props}>{children}</List>,
            ol: ({ children, ...props }) => (
              <List ordered {...props}>
                {children}
              </List>
            ),
            li: ({ children, ...props }) => <ListItem {...props}>{children}</ListItem>,
            // Inline elements
            a: ({ children, ...props }) => <Link {...props}>{children}</Link>,
            code: ({ children }) => <InlineCode>{children}</InlineCode>,
            img: ({ ...props }) => <Image {...props} />,
            audio: ({ node, ...props }: any) => {
              // Render audio with a copy-transcript button.
              // Copy logic: try to find a nearby transcript text (next sibling or parent text).
              const handleCopy = (e: React.MouseEvent) => {
                e.stopPropagation();
                try {
                  // Find the button's nearest parent element that contains the audio
                  const btn = e.currentTarget as HTMLElement;
                  const parent = btn.closest(".memo-audio-wrapper") as HTMLElement | null;
                  let textToCopy = "";
                  if (parent) {
                    // Prefer next element sibling (commonly transcript is placed after audio)
                    const next = parent.nextElementSibling as HTMLElement | null;
                    if (next && next.innerText.trim()) {
                      textToCopy = next.innerText.trim();
                    } else {
                      // Fallback: search within parent for elements with 'transcript' class
                      const transcriptEl = parent.querySelector(".transcript, .audio-transcript") as HTMLElement | null;
                      if (transcriptEl && transcriptEl.innerText.trim()) {
                        textToCopy = transcriptEl.innerText.trim();
                      } else {
                        // As last resort, use parent's innerText minus any button text
                        textToCopy = parent.innerText.replace(/\n|Copy Transcript/g, "").trim();
                      }
                    }
                  }
                  if (textToCopy) {
                    navigator.clipboard.writeText(textToCopy);
                    toast.success("Copied transcript");
                  }
                } catch (err) {
                  console.error(err);
                }
              };

              return (
                <div className="memo-audio-wrapper flex items-center gap-2">
                  <audio {...props} controls />
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 px-2 py-1 text-xs rounded bg-muted text-muted-foreground hover:bg-muted/80"
                    onClick={handleCopy}
                    aria-label="Copy transcript"
                  >
                    Copy
                  </button>
                </div>
              );
            },
            // Code blocks
            pre: CodeBlock,
            // Tables
            table: ({ children }) => <Table>{children}</Table>,
            thead: ({ children }) => <TableHead>{children}</TableHead>,
            tbody: ({ children }) => <TableBody>{children}</TableBody>,
            tr: ({ children }) => <TableRow>{children}</TableRow>,
            th: ({ children, ...props }) => <TableHeaderCell {...props}>{children}</TableHeaderCell>,
            td: ({ children, ...props }) => <TableCell {...props}>{children}</TableCell>,
          }}
        >
          {content}
        </ReactMarkdown>
        {showCompactMode === "ALL" && (
          <div
            className={cn(
              "absolute inset-x-0 bottom-0 pointer-events-none",
              COMPACT_MODE_CONFIG.gradientHeight,
              "bg-linear-to-t from-background from-0% via-background/60 via-40% to-transparent to-100%",
            )}
          />
        )}
      </div>
      {showCompactMode !== undefined && (
        <div className="relative w-full mt-2">
          <button
            type="button"
            className="group inline-flex items-center gap-1 px-2 py-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
            onClick={toggleCompactMode}
          >
            <span>{compactLabel}</span>
            {showCompactMode === "ALL" ? <ChevronDown className="w-3 h-3" /> : <ChevronUp className="w-3 h-3" />}
          </button>
        </div>
      )}
    </div>
  );
};

export default memo(MemoContent);
