import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import React, {
  createContext,
  ReactElement,
  useContext,
  useEffect,
  useState,
} from 'react';

// Supported auth providers
type AuthProvider = 'password' | 'google' | 'linkedin' | 'github';

interface WorkspaceContextProps {
  domain: string;
  logo: ReactElement;
  authentication: AuthConfig;
}

interface AuthConfig {
  signIn: AuthMethodConfig;
  signUp: AuthMethodConfig;
  passwordRules?: PasswordRules;
}

interface AuthMethodConfig {
  enable: boolean;
  providers: Record<AuthProvider, boolean>;
}

interface PasswordRules {
  minLength?: number;
  requireUppercase?: boolean;
  requireLowercase?: boolean;
  requireNumber?: boolean;
  requireSpecialChar?: boolean;
}

// Hardcoded workspace configurations
const workspaceConfigs: Record<string, WorkspaceContextProps> = {
  'voice.yuukiai.com': {
    domain: 'voice.yuukiai.com',
    logo: (
      <div>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          className="h-8"
          viewBox="0 0 150 40"
        >
          <path
            d="M0 10C0 4.47715 4.47715 0 10 0H30C35.5228 0 40 4.47715 40 10V30C40 35.5228 35.5228 40 30 40H10C4.47715 40 0 35.5228 0 30V10Z"
            fill="url(#paint0_linear_1058_6443)"
          ></path>
          <path
            fill-rule="evenodd"
            clip-rule="evenodd"
            d="M29.7164 10.3879C30.8628 11.4467 30.9369 13.2377 29.8819 14.3883L19.8136 25.0269L9.79474 14.441C8.71077 13.3179 8.7392 11.5255 9.85823 10.4376C10.9773 9.34973 12.7632 9.37826 13.8471 10.5014L19.9659 16.8411L25.7304 10.5541C26.7854 9.40348 28.5699 9.32907 29.7164 10.3879Z"
            fill="#DDDDDD"
          ></path>
          <path
            d="M22.7627 31.0508C22.7627 32.8751 21.6312 34 19.8136 34C17.9959 34 16.8644 32.8751 16.8644 31.0508C16.8644 31.0508 16.8644 24.0393 16.8644 21.9108L19.8136 23.1864L22.7627 21.9107C22.7627 24.5308 22.7627 30.385 22.7627 31.0508Z"
            fill="#DDDDDD"
          ></path>
          <path
            d="M23.257 8.29957C23.257 10.1238 21.7835 11.6026 19.9659 11.6026C18.1483 11.6026 16.6748 10.1238 16.6748 8.29957C16.6748 6.47535 18.1483 4.99652 19.9659 4.99652C21.7835 4.99652 23.257 6.47535 23.257 8.29957Z"
            fill="#DDDDDD"
          ></path>
          <path d="M71.9398 11.0881C71.8208 10.366 71.4136 9.72834 70.8221 9.30583C69.6582 8.40702 68.0104 8.63748 67.0501 9.82437L60.8584 16.0546L60.8084 16.1083C60.697 16.2428 60.5664 16.3196 60.4397 16.3196C60.3129 16.3196 60.1823 16.2466 60.0709 16.1083L53.8561 9.84742C53.4681 9.2943 52.8881 8.92172 52.2083 8.80265C51.4823 8.67205 50.7179 8.85258 50.1149 9.29815C49.5003 9.75139 49.1124 10.4044 49.0202 11.1342C48.9318 11.8371 49.1354 12.5323 49.5925 13.0893L57.6626 21.893C57.701 21.9391 57.7548 22.0197 57.7548 22.1695V31.2729C57.7548 32.7785 58.9801 34 60.4819 34C61.9838 34 63.2091 32.8016 63.2091 31.2729V22.1695C63.2091 22.1427 63.2168 22.0389 63.3205 21.8738L71.4174 13.1277L71.4751 13.0585C71.8899 12.4862 72.0589 11.7872 71.9437 11.0919L71.9398 11.0881Z"></path>
          <path d="M92.9773 8.76423C91.4255 8.76423 90.2117 9.96265 90.2117 11.4914V22.8993C90.2117 25.9722 87.715 28.4689 84.6422 28.4689C81.5694 28.4689 79.0727 25.9722 79.0727 22.8993V11.4914C79.0727 9.98569 77.832 8.76423 76.3071 8.76423C74.7822 8.76423 73.5799 9.96265 73.5799 11.4914V22.8993C73.5799 28.999 78.5426 33.9616 84.6422 33.9616C90.7418 33.9616 95.6622 29.0028 95.7045 22.8993V11.4914C95.7045 9.96265 94.506 8.76423 92.9773 8.76423Z"></path>
          <path d="M117.56 8.76423C116.008 8.76423 114.795 9.96265 114.795 11.4914V22.8993C114.795 25.9722 112.298 28.4689 109.225 28.4689C106.152 28.4689 103.656 25.9722 103.656 22.8993V11.4914C103.656 9.98569 102.415 8.76423 100.89 8.76423C99.365 8.76423 98.1628 9.96265 98.1628 11.4914V22.8993C98.1628 28.999 103.125 33.9616 109.225 33.9616C115.325 33.9616 120.287 28.999 120.287 22.8993V11.4914C120.287 9.96265 119.089 8.76423 117.56 8.76423Z"></path>
          <path d="M142.082 29.4752L134.86 20.0416C134.726 19.8495 134.734 19.5845 134.868 19.427L141.901 13.5771L141.966 13.5156C142.976 12.4517 142.969 10.6963 141.951 9.67841C140.895 8.62211 139.182 8.61443 138.118 9.6592L128.722 17.9405L128.688 17.9751C128.315 18.3477 128.073 18.4168 128.05 18.4283C128.008 18.3938 127.885 18.171 127.885 17.6371V11.5682C127.885 10.0625 126.663 8.84105 125.158 8.84105C123.652 8.84105 122.431 10.0664 122.431 11.5682V31.2344C122.431 31.9642 122.73 32.6748 123.256 33.1819C123.756 33.6658 124.405 33.927 125.066 33.927C125.096 33.927 125.127 33.927 125.158 33.927C126.687 33.927 127.885 32.7286 127.885 31.1999V26.629C127.885 26.5599 127.9 26.4331 128.012 26.3217L130.389 23.9057C130.512 23.7828 130.647 23.7636 130.735 23.7674C130.869 23.7751 131 23.8442 131.081 23.9441L137.918 32.8976C138.421 33.5506 139.174 33.927 139.984 33.927C140.63 33.927 141.225 33.7081 141.705 33.2894C142.277 32.8362 142.642 32.1601 142.708 31.4303C142.773 30.7082 142.546 30.0168 142.078 29.4829L142.082 29.4752Z"></path>
          <path d="M147.636 8.87946C146.107 8.87946 144.909 10.0779 144.909 11.6066V31.2344C144.909 32.7632 146.107 33.9616 147.636 33.9616C149.164 33.9616 150.401 32.7363 150.401 31.2344V11.6066C150.401 10.1009 149.161 8.87946 147.636 8.87946Z"></path>
          <path d="M60.674 5C58.7265 5 57.1402 6.58636 57.1402 8.53378C57.1402 10.4812 58.7265 12.0676 60.674 12.0676C62.6214 12.0676 64.2077 10.4812 64.2077 8.53378C64.2077 6.58636 62.6214 5 60.674 5Z"></path>
          <defs>
            <linearGradient
              id="paint0_linear_1058_6443"
              x1="7"
              y1="-8.19564e-07"
              x2="33.5"
              y2="40"
              gradientUnits="userSpaceOnUse"
            >
              <stop stop-color="#7A5AF5"></stop>
              <stop offset="1" stop-color="#382BF0"></stop>
            </linearGradient>
          </defs>
        </svg>
      </div>
    ),

    authentication: {
      signIn: {
        enable: true,
        providers: {
          password: false,
          google: false,
          linkedin: false,
          github: false,
        },
      },
      signUp: {
        enable: false,
        providers: {
          password: false,
          google: false,
          linkedin: false,
          github: false,
        },
      },
      passwordRules: {
        minLength: 8,
        requireUppercase: true,
        requireNumber: true,
        requireSpecialChar: true,
      },
    },
  },
  'voice.beanbag.ai': {
    domain: 'voice.beanbag.ai',
    logo: (
      <img
        alt="voice.beanbag.ai"
        className={'h-9 w-fit px-1'}
        src="https://www.beanbag.ai/static/media/darkLogo.040cbcdb4409413d0838.png"
      />
    ),

    authentication: {
      signIn: {
        enable: true,
        providers: {
          password: false,
          google: false,
          linkedin: false,
          github: false,
        },
      },
      signUp: {
        enable: false,
        providers: {
          password: false,
          google: false,
          linkedin: false,
          github: false,
        },
      },
    },
  },
};

// Default fallback workspace config
const defaultWorkspace: WorkspaceContextProps = {
  domain: 'rapida.ai',
  logo: (
    <div className="flex items-center shrink-0 space-x-1.5 ml-1 text-blue-600 dark:text-blue-500">
      <RapidaIcon className="h-8 w-8" />
      <RapidaTextIcon className="h-6" />
    </div>
  ),
  authentication: {
    signIn: {
      enable: true,
      providers: {
        password: true,
        google: true,
        linkedin: true,
        github: true,
      },
    },
    signUp: {
      enable: true,
      providers: {
        password: true,
        google: true,
        linkedin: true,
        github: true,
      },
    },
  },
};

const WorkspaceContext = createContext<WorkspaceContextProps>(defaultWorkspace);
export const useWorkspace = () => useContext(WorkspaceContext);
export const WorkspaceProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [workspaceInfo, setWorkspaceInfo] =
    useState<WorkspaceContextProps>(defaultWorkspace);
  useEffect(() => {
    const getCurrentDomain = () => {
      // Use window.location.hostname, falling back to a default if not available
      return typeof window !== 'undefined'
        ? window.location.hostname
        : 'default';
    };

    const currentDomain = getCurrentDomain();

    if (currentDomain in workspaceConfigs) {
      setWorkspaceInfo(workspaceConfigs[currentDomain]);
    } else {
      setWorkspaceInfo(defaultWorkspace);
    }
  }, []);

  return (
    <WorkspaceContext.Provider value={workspaceInfo}>
      {children}
    </WorkspaceContext.Provider>
  );
};
