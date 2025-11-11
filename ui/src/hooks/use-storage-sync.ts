import { useEffect } from 'react';

export const serializeProto = (protoObj: any) => protoObj.serializeBinary();
export const deserializeProto = (ProtoClass: any, binaryData: Uint8Array) =>
  ProtoClass.deserializeBinary(binaryData);

export const LOCAL_STORAGE_PROVIDERS = '__rai__pms';
export const LOCAL_STORAGE_TOOLS = '__rai__ts';
export const LOCAL_STORAGE_PROVIDER_CREDENTIALS = '__rai__pcs';
export const LOCAL_STORAGE_TOOL_CREDENTIALS = '__rai__tls';

export const useLocalStorageSync = (
  key: string,
  setter: (value: any) => void,
  ProtoClass: any,
) => {
  useEffect(() => {
    const handleStorageChange = () => {
      const savedData = localStorage.getItem(key);
      if (savedData) {
        setter(
          JSON.parse(savedData).map((data: any) =>
            deserializeProto(ProtoClass, new Uint8Array(data)),
          ),
        );
      }
    };

    handleStorageChange();
    window.addEventListener('storage', handleStorageChange);
    return () => {
      window.removeEventListener('storage', handleStorageChange);
    };
  }, [key, setter, ProtoClass]);
};
