import React, { type FC } from "react";
import { Text } from "ink";
import type { UIMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";

interface UserTextMessageProps {
  message: UIMessage;
  continuation?: boolean;
}

const UserTextMessage: FC<UserTextMessageProps> = ({
  message,
  continuation = false,
}) => {
  return (
    <MessageRow
      markerColor="cyan"
      label={
        continuation ? null : (
          <Text color="cyan" bold>
            You
          </Text>
        )
      }
    >
      <Text>{message.text}</Text>
    </MessageRow>
  );
};

export default UserTextMessage;
