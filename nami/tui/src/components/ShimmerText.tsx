import React, {
  type ComponentProps,
  type FC,
  useEffect,
  useState,
} from "react";
import { Box, Text } from "silvery";

interface ShimmerTextProps {
  text: string;
  activeColor?: ComponentProps<typeof Text>["color"];
  inactiveColor?: ComponentProps<typeof Text>["color"];
}

const ShimmerText: FC<ShimmerTextProps> = ({
  text,
  activeColor = "$info",
  inactiveColor = "$muted",
}) => {
  const [frame, setFrame] = useState(0);

  useEffect(() => {
    // The range of frame is [0, text.length + shimmerWidth + pause]
    // shimmerWidth is 4, pause is 6.
    const totalFrames = text.length + 10;
    const timer = setInterval(() => {
      setFrame((f) => (f + 1) % totalFrames);
    }, 60);

    return () => clearInterval(timer);
  }, [text.length]);

  return (
    <Box flexDirection="row" minWidth={0}>
      {text.split("").map((char, i) => {
        // We want the shimmer to move from left to right.
        // A character is highlighted if it's within the 'shimmer' window.
        const shimmerWidth = 4;
        const dist = i - frame + shimmerWidth;
        const isHighlight = dist >= 0 && dist < shimmerWidth;

        // Optionally, we could have different levels of brightness,
        // but simple toggle between a light accent and muted is a good start.
        return (
          <Text
            key={i}
            color={isHighlight ? activeColor : inactiveColor}
            italic
          >
            {char}
          </Text>
        );
      })}
    </Box>
  );
};

export default ShimmerText;
