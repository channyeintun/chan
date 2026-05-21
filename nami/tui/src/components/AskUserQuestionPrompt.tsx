import React, { type FC, useMemo, useState } from "react";
import { Box, Text, useInput } from "silvery";
import type {
  UIAskUserQuestionAnswer,
  UIAskUserQuestionRequest,
} from "../hooks/useEvents.js";

interface AskUserQuestionPromptProps {
  request: UIAskUserQuestionRequest;
  onSubmit: (
    status: "answered" | "declined" | "cancelled",
    answers: UIAskUserQuestionAnswer[],
  ) => void;
}

const AskUserQuestionPrompt: FC<AskUserQuestionPromptProps> = ({
  request,
  onSubmit,
}) => {
  const initialAnswers = useMemo(
    () =>
      request.questions.map((question) => {
        const recommended = question.options.find(
          (option) => option.recommended,
        );
        const initialValue = recommended?.value ?? question.options[0]?.value;
        return {
          header: question.header,
          selectedValues: initialValue ? [initialValue] : [],
          freeformText: "",
          rawAnswer: initialValue ?? "",
        };
      }),
    [request.questions],
  );
  const [questionIndex, setQuestionIndex] = useState(0);
  const [answers, setAnswers] =
    useState<UIAskUserQuestionAnswer[]>(initialAnswers);
  const [optionIndex, setOptionIndex] = useState(0);
  const [freeformDraft, setFreeformDraft] = useState("");

  const currentQuestion = request.questions[questionIndex];
  const currentAnswer = answers[questionIndex] ?? initialAnswers[questionIndex];

  if (!currentQuestion || !currentAnswer) {
    return null;
  }

  const persistCurrentAnswer = () => {
    const freeformText = currentQuestion.allowFreeform
      ? freeformDraft.trim()
      : "";
    setAnswers((existing) =>
      existing.map((answer, index) => {
        if (index !== questionIndex) {
          return answer;
        }
        const rawAnswer = [...answer.selectedValues, freeformText]
          .filter((value) => value.length > 0)
          .join(", ");
        return {
          ...answer,
          freeformText,
          rawAnswer,
        };
      }),
    );
  };

  const buildFinalAnswers = () =>
    answers.map((answer, index) => {
      if (index !== questionIndex) {
        return answer;
      }
      const freeformText = currentQuestion.allowFreeform
        ? freeformDraft.trim()
        : answer.freeformText;
      const rawAnswer = [...answer.selectedValues, freeformText]
        .filter((value) => value.length > 0)
        .join(", ");
      return {
        ...answer,
        freeformText,
        rawAnswer,
      };
    });

  const toggleSelection = () => {
    const option = currentQuestion.options[optionIndex];
    if (!option) {
      return;
    }
    setAnswers((existing) =>
      existing.map((answer, index) => {
        if (index !== questionIndex) {
          return answer;
        }
        if (currentQuestion.multiSelect) {
          const selectedValues = answer.selectedValues.includes(option.value)
            ? answer.selectedValues.filter((value) => value !== option.value)
            : [...answer.selectedValues, option.value];
          return {
            ...answer,
            selectedValues,
          };
        }
        return {
          ...answer,
          selectedValues: [option.value],
        };
      }),
    );
  };

  useInput((input, key) => {
    if (key.escape) {
      onSubmit("cancelled", []);
      return;
    }
    if (key.ctrl && input === "d") {
      onSubmit("declined", []);
      return;
    }

    const text = key.text ?? input;
    if (
      currentQuestion.allowFreeform &&
      typeof text === "string" &&
      text.length === 1 &&
      !key.ctrl &&
      !key.meta &&
      !key.return &&
      !key.upArrow &&
      !key.downArrow
    ) {
      setFreeformDraft((value) => value + text);
      return;
    }
    if (currentQuestion.allowFreeform && key.backspace) {
      setFreeformDraft((value) =>
        value.slice(0, Math.max(0, value.length - 1)),
      );
      return;
    }

    if (key.upArrow) {
      if (currentQuestion.options.length > 0) {
        setOptionIndex((index) =>
          index <= 0 ? currentQuestion.options.length - 1 : index - 1,
        );
      }
      return;
    }
    if (key.downArrow) {
      if (currentQuestion.options.length > 0) {
        setOptionIndex((index) => (index + 1) % currentQuestion.options.length);
      }
      return;
    }
    if (input === " " && currentQuestion.options.length > 0) {
      toggleSelection();
      return;
    }
    if (!currentQuestion.multiSelect && currentQuestion.options.length > 0) {
      const shortcut = input?.toLowerCase() ?? "";
      const numericIndex = Number.parseInt(shortcut, 10);
      if (
        !Number.isNaN(numericIndex) &&
        numericIndex >= 1 &&
        numericIndex <= currentQuestion.options.length
      ) {
        setOptionIndex(numericIndex - 1);
        setAnswers((existing) =>
          existing.map((answer, index) =>
            index === questionIndex
              ? {
                  ...answer,
                  selectedValues: [
                    currentQuestion.options[numericIndex - 1].value,
                  ],
                }
              : answer,
          ),
        );
        return;
      }
    }
    if (key.return) {
      persistCurrentAnswer();
      if (questionIndex >= request.questions.length - 1) {
        onSubmit("answered", buildFinalAnswers());
        return;
      }
      const nextIndex = questionIndex + 1;
      setQuestionIndex(nextIndex);
      setOptionIndex(0);
      setFreeformDraft(
        answers[nextIndex]?.freeformText ??
          initialAnswers[nextIndex]?.freeformText ??
          "",
      );
    }
  });

  return (
    <Box
      flexDirection="column"
      flexGrow={1}
      flexShrink={1}
      minWidth={0}
      minHeight={0}
      backgroundColor="$popover-bg"
      borderStyle="single"
      borderColor="$inputborder"
      overflow="hidden"
      paddingX={2}
      paddingY={1}
    >
      <Text bold color="$primary">
        Clarification Required
      </Text>
      <Box marginTop={1} flexDirection="column" minWidth={0}>
        <Text>{`Question ${questionIndex + 1} of ${request.questions.length}`}</Text>
        <Text bold>{currentQuestion.question}</Text>
        <Text color="$muted">Header: {currentQuestion.header}</Text>
      </Box>

      {currentQuestion.options.length > 0 ? (
        <Box marginTop={1} flexDirection="column" minWidth={0}>
          {currentQuestion.options.map((option, index) => {
            const selected = currentAnswer.selectedValues.includes(
              option.value,
            );
            const cursor = index === optionIndex;
            return (
              <Box key={option.value} flexDirection="column" marginBottom={1}>
                <Text color={cursor ? "$primary" : "$fg"} bold={cursor}>
                  {cursor ? "›" : " "} {selected ? "[x]" : "[ ]"} {index + 1}.{" "}
                  {option.label}
                </Text>
                {option.description ? (
                  <Text color="$muted"> {option.description}</Text>
                ) : null}
              </Box>
            );
          })}
        </Box>
      ) : null}

      {currentQuestion.allowFreeform ? (
        <Box marginTop={1} flexDirection="column" minWidth={0}>
          <Text color="$muted">Optional note</Text>
          <Text>{freeformDraft || "Type to add freeform context..."}</Text>
        </Box>
      ) : null}

      <Box marginTop={1} flexDirection="column" flexShrink={0}>
        <Text dimColor>
          Enter next/submit · Up/Down move · Space toggle · Ctrl+D decline · Esc
          cancel
        </Text>
      </Box>
    </Box>
  );
};

export default AskUserQuestionPrompt;
