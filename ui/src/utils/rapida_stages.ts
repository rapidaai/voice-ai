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
export type RapidaStage =
  | 'user-authentication'
  | 'audio-transcription'
  | 'assistant-identificaion'
  | 'undefined'
  | 'query-formulation'
  | 'information-retrieval'
  | 'document-retrieval'
  | 'context-augmentation'
  | 'text-generation'
  | 'output-evaluation';

export const AuthenticationStage: RapidaStage = 'user-authentication';
export const TranscriptionStage: RapidaStage = 'audio-transcription';
export const AssistantIdentificationStage: RapidaStage =
  'assistant-identificaion';
export const UndefinedStage: RapidaStage = 'undefined';
export const QueryFormulationStage: RapidaStage = 'query-formulation';
export const InformationRetrievalStage: RapidaStage = 'information-retrieval';
export const DocumentRetrievalStage: RapidaStage = 'document-retrieval';
export const ContextAugmentationStage: RapidaStage = 'context-augmentation';
export const TextGenerationStage: RapidaStage = 'text-generation';
export const OutputEvaluationStage: RapidaStage = 'output-evaluation';

// Function to get the string value of a RapidaStage
export function getRapidaStageString(stage: RapidaStage): string {
  return stage;
}

// Function to return the corresponding RapidaStage for a given string,
// or UndefinedStage if the string does not match any stage.
export function fromStageStr(label: string): RapidaStage {
  switch (label.toLowerCase()) {
    case 'user-authentication':
      return AuthenticationStage;
    case 'audio-transcription':
      return TranscriptionStage;
    case 'assistant-identificaion':
      return AssistantIdentificationStage;
    case 'query-formulation':
      return QueryFormulationStage;
    case 'information-retrieval':
      return InformationRetrievalStage;
    case 'document-retrieval':
      return DocumentRetrievalStage;
    case 'context-augmentation':
      return ContextAugmentationStage;
    case 'text-generation':
      return TextGenerationStage;
    case 'output-evaluation':
      return OutputEvaluationStage;
    default:
      console.warn(
        `${label} is not a supported stage. Supported stages are: 'authentication', 'transcription', 'assistant', 'query-formulation', 'information-retrieval', 'document-retrieval', 'context-augmentation', 'text-generation', and 'output-evaluation'.`,
      );
      return UndefinedStage;
  }
}
