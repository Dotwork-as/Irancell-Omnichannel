'use client';

import { useEffect, useState } from 'react';

const DEFAULT_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()';

type ScrambleTextProps = {
  text: string;
  className?: string;
  duration?: number;
  chars?: string;
  onDone?: () => void;
};

export default function ScrambleText({
  text,
  className = '',
  duration = 700,
  chars = DEFAULT_CHARS,
  onDone,
}: ScrambleTextProps) {
  const [output, setOutput] = useState(text);

  useEffect(() => {
    let startTime: number | null = null;

    const step = (timestamp: number) => {
      if (!startTime) startTime = timestamp;
      const elapsed = timestamp - startTime;
      const progress = Math.min(elapsed / duration, 1);

      const scrambled = text
        .split('')
        .map((char, i) => {
          if (char === ' ') return ' ';
          const revealThreshold = (i + 1) / text.length;
          return progress >= revealThreshold
            ? char
            : chars[Math.floor(Math.random() * chars.length)];
        })
        .join('');

      setOutput(scrambled);

      if (progress < 1) {
        requestAnimationFrame(step);
      } else {
        setOutput(text);
        onDone?.();
      }
    };

    requestAnimationFrame(step);

    return () => {};
  }, [text, duration, chars, onDone]);

  return <span className={className}>{output}</span>;
}
