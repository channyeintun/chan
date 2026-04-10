import React, { type FC, useMemo } from "react";
import { Box } from "ink";
import type {
  UIMessage,
  UIToolCall,
  UITranscriptEntry,
} from "../hooks/useEvents.js";
import GroupedToolCalls, { type ToolCallGroup } from "./GroupedToolCalls.js";
import ToolProgress from "./ToolProgress.js";
import AssistantTextMessage from "./messages/AssistantTextMessage.js";
import AssistantThinkingMessage from "./messages/AssistantThinkingMessage.js";
import StreamingAssistantMessage from "./messages/StreamingAssistantMessage.js";
import UserTextMessage from "./messages/UserTextMessage.js";

interface StreamOutputProps {
  messages: UIMessage[];
  toolCalls: UIToolCall[];
  transcript: UITranscriptEntry[];
  liveText: string;
  liveThinkingText: string;
  isStreaming: boolean;
  model: string;
}

type TranscriptBlock =
  | { kind: "message"; message: UIMessage; continuation: boolean }
  | { kind: "tool_call"; toolCall: UIToolCall }
  | { kind: "tool_group"; group: ToolCallGroup };

const StreamOutput: FC<StreamOutputProps> = ({
  messages,
  toolCalls,
  transcript,
  liveText,
  liveThinkingText,
  isStreaming,
  model,
}) => {
  const messageById = useMemo(
    () => new Map(messages.map((message) => [message.id, message])),
    [messages],
  );
  const toolCallById = useMemo(
    () => new Map(toolCalls.map((toolCall) => [toolCall.id, toolCall])),
    [toolCalls],
  );
  const transcriptBlocks = useMemo(
    () => buildTranscriptBlocks(transcript, messageById, toolCallById),
    [messageById, toolCallById, transcript],
  );

  if (
    transcript.length === 0 &&
    !liveText &&
    !liveThinkingText &&
    !isStreaming
  ) {
    return null;
  }

  return (
    <Box flexDirection="column" paddingLeft={1} marginTop={1}>
      {transcriptBlocks.map((block) => {
        if (block.kind === "tool_group") {
          return <GroupedToolCalls key={block.group.id} group={block.group} />;
        }

        if (block.kind === "tool_call") {
          return (
            <ToolProgress key={block.toolCall.id} toolCall={block.toolCall} />
          );
        }

        return block.message.role === "assistant" ? (
          <AssistantTextMessage
            key={block.message.id}
            message={block.message}
            continuation={block.continuation}
          />
        ) : (
          <UserTextMessage
            key={block.message.id}
            message={block.message}
            continuation={block.continuation}
          />
        );
      })}

      {isStreaming && liveThinkingText && !liveText ? (
        <AssistantThinkingMessage text={liveThinkingText} model={model} />
      ) : null}

      {isStreaming && (Boolean(liveText) || !liveThinkingText) ? (
        <StreamingAssistantMessage
          text={liveText || undefined}
          model={model}
          statusLabel={
            liveText ? "Responding" : liveThinkingText ? "Thinking" : "Working"
          }
        />
      ) : null}
    </Box>
  );
};

export default StreamOutput;

function buildTranscriptBlocks(
  transcript: UITranscriptEntry[],
  messageById: Map<string, UIMessage>,
  toolCallById: Map<string, UIToolCall>,
): TranscriptBlock[] {
  const blocks: TranscriptBlock[] = [];
  let previousMessageRole: UIMessage["role"] | null = null;

  for (let index = 0; index < transcript.length; index += 1) {
    const entry = transcript[index];
    if (!entry) {
      continue;
    }

    if (entry.kind !== "tool_call") {
      const message = messageById.get(entry.id);
      if (!message) {
        continue;
      }

      blocks.push({
        kind: "message",
        message,
        continuation: previousMessageRole === message.role,
      });
      previousMessageRole = message.role;
      continue;
    }

    const run: UIToolCall[] = [];
    let cursor = index;
    while (
      cursor < transcript.length &&
      transcript[cursor]?.kind === "tool_call"
    ) {
      const toolCall = toolCallById.get(transcript[cursor]!.id);
      if (toolCall) {
        run.push(toolCall);
      }
      cursor += 1;
    }

    blocks.push(...buildToolBlocks(run));
    previousMessageRole = null;
    index = cursor - 1;
  }

  return blocks;
}

function buildToolBlocks(toolCalls: UIToolCall[]): TranscriptBlock[] {
  const blocks: TranscriptBlock[] = [];

  for (let index = 0; index < toolCalls.length; index += 1) {
    const toolCall = toolCalls[index];
    const groupKind = toolGroupKind(toolCall);

    if (groupKind !== "read_search") {
      blocks.push({ kind: "tool_call", toolCall });
      continue;
    }

    const grouped: UIToolCall[] = [toolCall];
    let cursor = index + 1;
    while (
      cursor < toolCalls.length &&
      toolGroupKind(toolCalls[cursor]!) === groupKind
    ) {
      grouped.push(toolCalls[cursor]!);
      cursor += 1;
    }

    if (grouped.length >= 2) {
      blocks.push({
        kind: "tool_group",
        group: {
          id: `tool-group-${grouped[0]!.id}-${grouped[grouped.length - 1]!.id}`,
          kind: "read_search",
          toolCalls: grouped,
        },
      });
      index = cursor - 1;
      continue;
    }

    blocks.push({ kind: "tool_call", toolCall });
  }

  return blocks;
}

function toolGroupKind(toolCall: UIToolCall): ToolCallGroup["kind"] | null {
  switch (toolCall.name) {
    case "file_read":
    case "grep":
    case "glob":
    case "web_search":
    case "web_fetch":
    case "git":
      return "read_search";
    default:
      return null;
  }
}
