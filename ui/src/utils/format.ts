/*
* Formats a number with comma separators.
 formatNumber(1234567) will return '1,234,567'
 formatNumber(1234567.89) will return '1,234,567.89'
*/
export const formatNumber = (num: number | string) => {
  if (!num) return num;
  const parts = num.toString().split('.');
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return parts.join('.');
};

export const formatHumanReadableNumber = (num: number | string): string => {
  // Convert the input to a number if it's a string
  const number = typeof num === 'string' ? parseFloat(num) : num;

  // Check if the input is a valid number
  if (isNaN(number)) {
    return 'Invalid number';
  }

  // Define thresholds for different number ranges
  const thresholds = [
    { value: 1e9, suffix: 'B' }, // Billion
    { value: 1e6, suffix: 'M' }, // Million
    { value: 1e3, suffix: 'K' }, // Thousand
  ];

  // Find the appropriate threshold and format the number
  for (const threshold of thresholds) {
    if (number >= threshold.value) {
      const formattedNumber = (number / threshold.value).toFixed(1);
      return `${formattedNumber}${threshold.suffix}`;
    }
  }

  // For numbers less than 1000, return the number itself, formatted with commas
  return number.toLocaleString();
};

export const formatFileSize = (num: number) => {
  if (!num) return num;
  const units = ['', 'K', 'M', 'G', 'T', 'P'];
  let index = 0;
  while (num >= 1024 && index < units.length) {
    num = num / 1024;
    index++;
  }
  return `${num.toFixed(2)}${units[index]}B`;
};

export const formatTime = (num: number) => {
  if (!num) return num;
  const units = ['sec', 'min', 'h'];
  let index = 0;
  while (num >= 60 && index < units.length) {
    num = num / 60;
    index++;
  }
  return `${num.toFixed(2)} ${units[index]}`;
};
