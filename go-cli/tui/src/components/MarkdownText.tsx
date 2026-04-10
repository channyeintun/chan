import React, { type FC, useMemo } from "react";
import { Text } from "ink";
import { marked, type MarkedExtension } from "marked";
import { markedTerminal } from "marked-terminal";

marked.use(markedTerminal({ reflowText: true, tab: 2 }) as MarkedExtension);

interface MarkdownTextProps {
  text: string;
}

const MarkdownText: FC<MarkdownTextProps> = ({ text }) => {
  const rendered = useMemo(() => {
    const result = marked.parse(text);
    if (typeof result !== "string") return text;
    return result.replace(/\n+$/, "");
  }, [text]);

  return <Text>{rendered}</Text>;
};

export default MarkdownText;
