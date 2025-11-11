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

export type RapidaSource =
  | 'web-plugin'
  | 'debugger'
  | 'rapida-app'
  | 'node-sdk'
  | 'go-sdk'
  | 'typescript-sdk'
  | 'java-sdk'
  | 'php-sdk'
  | 'rust-sdk'
  | 'python-sdk';

export const WEB_PLUGIN_SOURCE: RapidaSource = 'web-plugin';
export const DEBUGGER_SOURCE: RapidaSource = 'debugger';
export const RAPIDA_APP_SOURCE: RapidaSource = 'rapida-app';

export const PYTHONSDK_SOURCE: RapidaSource = 'python-sdk';
export const NODESDK_SOURCE: RapidaSource = 'node-sdk';
export const GOSDK_SOURCE: RapidaSource = 'go-sdk';
export const TYPESCRIPTSDK_SOURCE: RapidaSource = 'typescript-sdk';
export const JAVASDK_SOURCE: RapidaSource = 'java-sdk';
export const PHPSDK_SOURCE: RapidaSource = 'php-sdk';
export const RUSTSDK_SOURCE: RapidaSource = 'rust-sdk';

// Get returns the string value of the RapidaSource
export function getRapidaSourceValue(source: RapidaSource): string {
  return source;
}

// FromStr returns the corresponding RapidaSource for a given string,
// or 'web-plugin' if the string does not match any source.
export function fromStr(label: string): RapidaSource {
  switch (label.toLowerCase()) {
    case 'web-plugin':
      return WEB_PLUGIN_SOURCE;
    case 'debugger':
      return DEBUGGER_SOURCE;
    case 'rapida-app':
      return RAPIDA_APP_SOURCE;
    case 'python-sdk':
      return PYTHONSDK_SOURCE;
    case 'node-sdk':
      return NODESDK_SOURCE;
    case 'go-sdk':
      return GOSDK_SOURCE;
    case 'typescript-sdk':
      return TYPESCRIPTSDK_SOURCE;
    case 'java-sdk':
      return JAVASDK_SOURCE;
    case 'php-sdk':
      return PHPSDK_SOURCE;
    case 'rust-sdk':
      return RUSTSDK_SOURCE;

    default:
      console.warn(
        "The source is not supported. Only 'web-plugin', 'debugger', 'rapida-app', and 'sdk' are allowed.",
      );
      return WEB_PLUGIN_SOURCE;
  }
}
