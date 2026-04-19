import React, { type FC } from "react";
import { Box, Text } from "silvery";

const NAMI_ASCII = `███╗   ██╗ █████╗ ███╗   ███╗██╗
████╗  ██║██╔══██╗████╗ ████║██║
██╔██╗ ██║███████║██╔████╔██║██║
██║╚██╗██║██╔══██║██║╚██╔╝██║██║
██║ ╚████║██║  ██║██║ ╚═╝ ██║██║
╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝     ╚═╝╚═╝`;

const ASCII_COLOR = "#00afff";

const WelcomeBanner: FC = () => {
  const lines = NAMI_ASCII.split("\n");

  return (
    <Box
      flexDirection="column"
      flexGrow={1}
      flexShrink={1}
      minHeight={0}
      marginTop={2}
      justifyContent="flex-start"
    >
      <Box flexDirection="column" paddingLeft={2}>
        {lines.map((line, index) => (
          <Text key={`nami-banner-${index}`} color={ASCII_COLOR}>
            {line}
          </Text>
        ))}
      </Box>
    </Box>
  );
};

export default WelcomeBanner;