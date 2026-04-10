import React, { type ComponentProps, type FC, type ReactNode } from "react";
import { Box, Text } from "ink";

interface MessageRowProps {
  children: ReactNode;
  marker?: string;
  markerColor?: ComponentProps<typeof Text>["color"];
  markerDim?: boolean;
  marginBottom?: number;
}

const DEFAULT_MARKER = "●";

const MessageRow: FC<MessageRowProps> = ({
  children,
  marker = DEFAULT_MARKER,
  markerColor,
  markerDim,
  marginBottom = 1,
}) => {
  return (
    <Box
      flexDirection="row"
      alignItems="flex-start"
      marginBottom={marginBottom}
    >
      <Box minWidth={2}>
        <Text color={markerColor} dimColor={markerDim}>
          {marker}
        </Text>
      </Box>
      <Box flexDirection="column" flexGrow={1}>
        {children}
      </Box>
    </Box>
  );
};

export default MessageRow;
