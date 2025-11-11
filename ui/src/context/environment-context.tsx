import React, { createContext, useContext } from 'react';

interface EnvironmentContextProps {
  isElectron: boolean;
}

const EnvironmentContext = createContext<EnvironmentContextProps>({
  isElectron: false,
});

export const useEnvironment = () => {
  return useContext(EnvironmentContext);
};

export const EnvironmentProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const isElectron = !!window.isElectron;

  return (
    <EnvironmentContext.Provider value={{ isElectron }}>
      {children}
    </EnvironmentContext.Provider>
  );
};
