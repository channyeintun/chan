import React, { type FC } from "react";
import { Box, Text } from "silvery";
import { DEFAULT_PROMPT_MARKER } from "../../constants/prompt.js";
import type { UIUserMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";

interface UserTextMessageProps {
  message: UIUserMessage;
  continuation?: boolean;
}

const UserTextMessage: FC<UserTextMessageProps> = ({
  message,
  continuation = false,
}) => {
  return (
    <MessageRow
      marker={DEFAULT_PROMPT_MARKER.trimEnd()}
      markerColor="$primary"
      label={null}
    >
      <Box width="100%" minWidth={0}>
        <Text wrap="wrap">{message.text}</Text>
      </Box>
    </MessageRow>
  );
};

export default UserTextMessage;
