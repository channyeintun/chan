import React, { type FC } from "react";
import { Text } from "ink";
import type { UIMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";

interface UserTextMessageProps {
  message: UIMessage;
}

const UserTextMessage: FC<UserTextMessageProps> = ({ message }) => {
  return (
    <MessageRow markerColor="cyan">
      <Text color="cyan" bold>
        You
      </Text>
      <Text>{message.text}</Text>
    </MessageRow>
  );
};

export default UserTextMessage;
