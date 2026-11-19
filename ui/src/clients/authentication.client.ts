import {
  AuthenticateUser,
  AuthorizeUser,
  ChangePassword,
  ChangePasswordRequest,
  CreatePassword,
  ForgotPassword,
  Github,
  Google,
  Linkedin,
  RegisterUser,
} from '@rapidaai/react';

import { ConnectionConfig } from '@rapidaai/react';
import { ApiAuth, withConnection } from './connection';

export type ChangeAccountPasswordParams = {
  currentPassword: string;
  password: string;
  auth: ApiAuth;
};

export const authenticateUser = withConnection(AuthenticateUser);
export const authorizeUser = withConnection(AuthorizeUser);
export const registerUser = withConnection(RegisterUser);
export const forgotPassword = withConnection(ForgotPassword);
export const createPassword = withConnection(CreatePassword);
export const changePassword = withConnection(ChangePassword);

export const googleAuth = withConnection(Google);
export const linkedinAuth = withConnection(Linkedin);
export const githubAuth = withConnection(Github);

export const changeAccountPassword = ({
  currentPassword,
  password,
  auth,
}: ChangeAccountPasswordParams) => {
  const request = new ChangePasswordRequest();
  request.setOldpassword(currentPassword);
  request.setPassword(password);

  return changePassword(
    request,
    ConnectionConfig.WithDebugger({
      authorization: auth.token,
      userId: auth.userId,
      projectId: auth.projectId,
    }),
  );
};
