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

export type RapidaRegion = 'ap' | 'us' | 'eu' | 'all';

export const AP_REGION: RapidaRegion = 'ap';
export const US_REGION: RapidaRegion = 'us';
export const EU_REGION: RapidaRegion = 'eu';
export const ALL_REGION: RapidaRegion = 'all';

// Get returns the string value of the RapidaRegion
export function getRapidaRegionValue(region: RapidaRegion): string {
  return region;
}

// FromStr returns the corresponding RapidaRegion for a given string,
// or 'all' if the string does not match any region.
export function fromStr(label: string): RapidaRegion {
  switch (label.toLowerCase()) {
    case 'ap':
      return AP_REGION;
    case 'us':
      return US_REGION;
    case 'eu':
      return EU_REGION;
    case 'all':
      return ALL_REGION;
    default:
      console.warn(
        "The region is not supported. Supported regions are 'ap', 'us', 'eu', and 'all'.",
      );
      return ALL_REGION;
  }
}
