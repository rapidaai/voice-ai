/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
import { Content } from '@rapidaai/react';

export enum ResponseContentType {
  TEXT_CONTENT = 'text',
  AUDIO_CONTENT = 'audio',
  IMAGE_CONTENT = 'image',
  MULTI_MEDIA_CONTENT = 'multi',
}

// Get returns the string value of the RapidaEnvironment
export function getResponseContentType(env: ResponseContentType): string {
  return env;
}

// FromStr returns the corresponding RapidaEnvironment for a given string,
// or DEVELOPMENT if the string does not match any environment.
export function fromStr(label: string): ResponseContentType {
  switch (label.toLowerCase()) {
    case ResponseContentType.TEXT_CONTENT:
      return ResponseContentType.TEXT_CONTENT;
    case ResponseContentType.AUDIO_CONTENT:
      return ResponseContentType.AUDIO_CONTENT;
    case ResponseContentType.IMAGE_CONTENT:
      return ResponseContentType.IMAGE_CONTENT;
    case ResponseContentType.MULTI_MEDIA_CONTENT:
      return ResponseContentType.MULTI_MEDIA_CONTENT;
    default:
      console.warn(
        'The content type is not supported. Only text, audio, image, multi',
      );
      return ResponseContentType.TEXT_CONTENT;
  }
}

type ResponseContentFormat = 'raw' | 'word' | 'url' | 'chunk';

export const TEXT_CONTENT_FORMAT_RAW: ResponseContentFormat = 'raw';
export const TEXT_CONTENT_FORMAT_WORD: ResponseContentFormat = 'word';

export const AUDIO_CONTENT_FORMAT_RAW: ResponseContentFormat = 'raw';
export const AUDIO_CONTENT_FORMAT_CHUNK: ResponseContentFormat = 'chunk';
export const AUDIO_CONTENT_FORMAT_URL: ResponseContentFormat = 'url';

export const IMAGE_CONTENT_FORMAT_RAW: ResponseContentFormat = 'raw';
export const IMAGE_CONTENT_FORMAT_URL: ResponseContentFormat = 'url';

export const MULTI_MEDIA_CONTENT_FORMAT_RAW: ResponseContentFormat = 'raw';
export const MULTI_MEDIA_CONTENT_FORMAT_URL: ResponseContentFormat = 'url';

/**
 *
 * @param str
 * @returns
 */
export const toTextContent = (str: string, format?: ResponseContentFormat) => {
  const cnt = new Content();
  if (format) cnt.setContentformat(format);
  else cnt.setContentformat(TEXT_CONTENT_FORMAT_RAW);
  cnt.setContenttype(ResponseContentType.TEXT_CONTENT);
  cnt.setContent(new TextEncoder().encode(str));
  return cnt;
};

export const toStreamAudioContent = (raw: Uint8Array | string): Content => {
  const cnt = new Content();
  cnt.setContentformat(AUDIO_CONTENT_FORMAT_CHUNK);
  cnt.setContenttype(ResponseContentType.AUDIO_CONTENT);
  cnt.setContent(raw);
  return cnt;
};
export const toContentText = (cnt?: Content[]): string => {
  if (!cnt) return '';

  const decodedTexts: string[] = [];
  cnt
    .filter(x => x.getContenttype() === 'text')
    .forEach(x => {
      const content = x.getContent();
      if (content instanceof Uint8Array) {
        // Handle Uint8Array
        try {
          const contentText = new TextDecoder().decode(content);
          decodedTexts.push(contentText);
        } catch (error) {
          console.error('Error decoding Uint8Array content:', error);
        }
      } else if (typeof content === 'string') {
        // Handle string directly
        decodedTexts.push(content);
      } else {
        console.error('Unexpected content type:', typeof content);
      }
    });

  return decodedTexts.join(' ');
};
