// import React, { useCallback, useContext, useEffect, useState } from 'react';
// import { useRapidaStore } from '@/hooks';
// import { useCredential } from '@/hooks/use-credential';
// import { SelectModel } from '@/app/pages/endpoint/actions/components/select-model';
// import { useCreateEndpointPageStore } from '@/hooks';
// import { useNavigate, useParams } from 'react-router-dom';
// import toast from 'react-hot-toast/headless';
// import { Helmet } from '@/app/components/Helmet';
// import { CreateEndpointContextProvider } from '@/context/create-endpoint-context';
// import { CreateEndpointContext } from '@/hooks/use-create-endpoint-page-store';
// import { Endpoint } from '@rapidaai/react';
// import {
//   HoverButton,
//   OutlineButton,
//   BlueBorderButton,
// } from '@/app/components/Form/Button';
// import { TabForm } from '@/app/components/Form/tab-form';
// import { SubtitleHeading } from '@/app/components/Heading/SubtitleHeading';
// import { TitleHeading } from '@/app/components/Heading/TitleHeading';
// import { EndpointIcon } from '@/app/components/Icon/Endpoint';
// import { EndpointIntegration } from '@/app/components/integration-document/endpoint-integration';
// import EndpointIdentifier from '@/app/pages/endpoint/actions/components/endpoint-identifier';
// import { EndpointVariable } from '@/app/pages/endpoint/actions/components/endpoint-variable';
// import { useAllProviderModels } from '@/hooks/use-model';

export function ConfigureEndpointPage() {
  //   const ctx = useCreateEndpointPageStore();
  /**
   * endpoint id
   */
  //   const { endpointId } = useParams();
  return <></>;
}
// /**
//  *
//  * @param props
//  * @returns
//  */
// function ConfigureEndpoint(props: { endpointId: string }) {
//   const [userId, token, projectId] = useCredential();
//   const [activeTab, setActiveTab] = useState('choose-model');
//   const [errorMessage, setErrorMessage] = useState('');

//   const [justCreatedEndpoint, setJustCreatedEndpoint] =
//     useState<Endpoint | null>(null);
//   const { providerModels } = useAllProviderModels();
//   const {
//     onGetEndpoint,
//     clear,
//     subType,
//     onChangeCurrentEndpoint,
//     onValidateEndpointProfile,
//     onValidateEndpointInstruction,
//     currentEndpoint,
//     onForkEndpoint,
//     onChangeName,
//     onChangeModel,
//     onChangeVisibility,
//   } = useContext(CreateEndpointContext);
//   /**
//    * show and hide loaders
//    */
//   const { loading, showLoader, hideLoader } = useRapidaStore();

//   /**
//    * get all the models when type change
//    */

//   let navigator = useNavigate();

//   useEffect(() => {
//     clear();
//     setJustCreatedEndpoint(null);
//   }, []);

//   /**
//    *
//    */
//   const afterGetEndpoint = useCallback(endpoint => {
//     onChangeCurrentEndpoint(endpoint);
//     onChangeName(`${endpoint.getName()}-copy`);
//     onChangeVisibility('private');
//     let mdl = providerModels.find(
//       x =>
//         x.getId() === endpoint.getEndpointprovidermodel()?.getProvidermodelid(),
//     );
//     if (mdl) onChangeModel(mdl);
//     hideLoader();
//   }, []);

//   /**
//    *
//    */
//   const getEndpoint = useCallback(
//     id => {
//       showLoader('overlay');
//       onGetEndpoint(id, projectId, userId, token, afterGetEndpoint, onError);
//     },
//     [props.endpointId],
//   );

//   /**
//    * on change of id everytime
//    */
//   useEffect(() => {
//     clear();
//     setJustCreatedEndpoint(null);
//     if (props.endpointId) {
//       getEndpoint(props.endpointId);
//     }
//   }, [props.endpointId]);

//   //
//   //

//   /**
//    *
//    */
//   const afterForkEndpoint = useCallback(
//     (e: Endpoint) => {
//       setJustCreatedEndpoint(e);
//       setActiveTab('integrate-endpoint');
//       toast.success('Your endpoint has been configured sucessfully.');
//     },
//     [justCreatedEndpoint],
//   );

//   /**
//    *
//    * @param error
//    */
//   const onError = error => {
//     hideLoader();
//     setErrorMessage(error);
//   };
//   /**
//    *
//    * @returns
//    */
//   const onconfigureendpoint = () => {
//     let endpointId = currentEndpoint?.getId();
//     if (!endpointId) {
//       setErrorMessage(
//         'Unable to configure endpoint, please try again in sometime.',
//       );
//       return;
//     }
//     showLoader('overlay');
//     onForkEndpoint(
//       endpointId,
//       projectId,
//       token,
//       userId,
//       afterForkEndpoint,
//       onError,
//     );
//   };

//   return (
//     <>
//       <Helmet title="Configure an endpoint"></Helmet>
//       <TabForm
//         activeTab={activeTab}
//         onChangeActiveTab={() => {}}
//         errorMessage={errorMessage}
//         headElement={
//           <div className="relative">
//             <TitleHeading className="font-semibold text-xl">
//               Configuring an Endpoint
//             </TitleHeading>
//             <SubtitleHeading className="font-medium opacity-70">
//               Customize and activate the endpoint to quick start
//             </SubtitleHeading>

//             <div className="absolute top-2 right-2">
//               <EndpointIcon
//                 className="w-12 h-12 opacity-20 text-green-600"
//                 strokeWidth="1.5"
//               />
//             </div>
//           </div>
//         }
//         form={[
//           {
//             name: 'Choose Model',
//             description: 'The model you want to use for your endpoint.',
//             code: 'choose-model',
//             body: (
//               <div className="space-y-6">
//                 <SelectModel readonly onlySubtype={subType} />
//                 <EndpointVariable />
//               </div>
//             ),
//             actions: [
//               <HoverButton
//                 onClick={() => navigator(-1)}
//                 className="text-blue-600 hover:text-gray-600 dark:hover:text-gray-300"
//               >
//                 Cancel
//               </HoverButton>,
//               <OutlineButton
//                 isLoading={loading}
//                 type="button"
//                 onClick={() => {
//                   const err = onValidateEndpointInstruction();
//                   if (err != null) {
//                     setErrorMessage(err);
//                     return;
//                   }
//                   setErrorMessage('');
//                   setActiveTab('define-endpoint');
//                 }}
//               >
//                 {/* Configure instruction */}
//                 Continue
//               </OutlineButton>,
//             ],
//           },
//           {
//             code: 'define-endpoint',
//             name: 'Define Endpoint Profile',
//             description:
//               'Provide the name, a brief description, and relevant tags.',
//             actions: [
//               <HoverButton
//                 onClick={() => navigator(-1)}
//                 className="text-blue-600 hover:text-gray-600 dark:hover:text-gray-300"
//               >
//                 Cancel
//               </HoverButton>,
//               <BlueBorderButton
//                 onClick={() => {
//                   setActiveTab('choose-model');
//                 }}
//               >
//                 Previous
//               </BlueBorderButton>,
//               <OutlineButton
//                 isLoading={loading}
//                 type="button"
//                 onClick={() => {
//                   setErrorMessage('');
//                   const err = onValidateEndpointProfile();
//                   if (err != null) {
//                     setErrorMessage(err);
//                     return;
//                   }
//                   onconfigureendpoint();
//                 }}
//               >
//                 Next
//               </OutlineButton>,
//             ],
//             body: <EndpointIdentifier />,
//           },
//           {
//             code: 'integrate-endpoint',
//             name: 'Integrate to your application',
//             description: 'Configure your application to use endpoint',
//             actions: [
//               <HoverButton
//                 onClick={() => {
//                   if (justCreatedEndpoint)
//                     navigator(
//                       `/deployment/endpoint/${justCreatedEndpoint.getId()}`,
//                     );
//                 }}
//                 className="text-blue-600 hover:text-gray-600 dark:hover:text-gray-300"
//               >
//                 Skip
//               </HoverButton>,
//               <OutlineButton
//                 isLoading={loading}
//                 type="button"
//                 onClick={() => {
//                   if (justCreatedEndpoint)
//                     navigator(
//                       `/deployment/endpoint/${justCreatedEndpoint.getId()}`,
//                     );
//                 }}
//               >
//                 Finish Setup
//               </OutlineButton>,
//             ],
//             body: justCreatedEndpoint ? (
//               <EndpointIntegration endpoint={justCreatedEndpoint} />
//             ) : (
//               <></>
//             ),
//           },
//         ]}
//       />
//     </>
//   );
// }
