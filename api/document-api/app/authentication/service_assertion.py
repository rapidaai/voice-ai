"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import base64
import hashlib
import hmac
import json
import time
from typing import Callable, Iterable

import jwt

from app.authentication.principal import (
    Identity,
    Principal,
    Proof,
    ProofReference,
    ServiceActor,
    Tenant,
)
from app.exceptions.authentication_exception import InvalidAuthorizationTokenException

ASSERTION_LIFETIME_SECONDS = 55
MAX_LIFETIME_SECONDS = 60
INTERNAL_SERVICE_ID = 1
INTERNAL_SERVICE_ISSUER = "internal"
SERVICE_AUTHENTICATION_LABEL = b"rapida/service-authentication/v1"


def derive_service_authentication_key(secret: str) -> bytes:
    """Derives the immutable service key from loader-validated configuration."""
    return hmac.digest(
        secret.encode("utf-8"), SERVICE_AUTHENTICATION_LABEL, "sha256"
    )


class ServiceAssertionIssuer:
    """Issues destination-bound request authentication signed by the shared key."""

    def __init__(
        self,
        destination: str,
        secret: str,
        now: Callable[[], float] = time.time,
    ):
        if (
            not destination
            or destination.strip() != destination
            or "\x00" in destination
        ):
            raise ValueError("invalid service assertion destination")
        key = derive_service_authentication_key(secret)
        self.destination = destination
        self.key = key
        self.key_fingerprint = base64.urlsafe_b64encode(
            hashlib.sha256(key).digest()
        ).decode("ascii").rstrip("=")
        self.now = now

    def issue(
        self,
        tenant: Tenant,
        subject: Identity = None,
        permissions: Iterable[str] = (),
    ) -> str:
        authorities = tuple(sorted(set(permissions)))
        proof = Proof(
            method="service_assertion",
            audience="rapida_internal",
            reference=ProofReference(type="signing_key", value=self.key_fingerprint),
        )
        asserted_subject = None
        if subject is None:
            subject = Identity(
                type="service",
                id=INTERNAL_SERVICE_ID,
                issuer=INTERNAL_SERVICE_ISSUER,
            )
            Principal(
                subject=subject,
                selectedTenant=tenant,
                proof=proof,
                authorities=authorities,
            )
        else:
            if subject.type == "service":
                raise ValueError("service subject must not be asserted")
            Principal(
                subject=subject,
                actor=ServiceActor(
                    id=INTERNAL_SERVICE_ID, issuer=INTERNAL_SERVICE_ISSUER
                ),
                selectedTenant=tenant,
                proof=proof,
                authorities=authorities,
            )
            asserted_subject = {"type": subject.type, "id": subject.id}

        authentication = {
            "tenant": {
                "organization_id": tenant.organization_id,
                "project_id": tenant.project_id,
            },
            "permissions": list(authorities),
        }
        if asserted_subject is not None:
            authentication["subject"] = asserted_subject
        claims = {
            "aud": self.destination,
            "exp": int(self.now()) + ASSERTION_LIFETIME_SECONDS,
            "authentication": authentication,
        }
        return jwt.encode(
            claims,
            self.key,
            algorithm="HS256",
            headers={"typ": "JWT"},
        )


class ServiceAssertionVerifier:
    """Verifies key-only assertions for exactly one destination service."""

    def __init__(
        self,
        destination: str,
        secret: str,
        now: Callable[[], float] = time.time,
    ):
        if (
            not destination
            or destination.strip() != destination
            or "\x00" in destination
        ):
            raise ValueError("invalid service assertion destination")
        key = derive_service_authentication_key(secret)
        self.destination = destination
        self.key = key
        self.key_fingerprint = base64.urlsafe_b64encode(
            hashlib.sha256(key).digest()
        ).decode("ascii").rstrip("=")
        self.now = now

    def verify(self, assertion: str) -> Principal:
        if not assertion or assertion.strip() != assertion:
            raise InvalidAuthorizationTokenException("invalid service assertion")
        header, claims = self.decode_closed_json(assertion)
        if set(header) != {"alg", "typ"} or header != {
            "alg": "HS256",
            "typ": "JWT",
        }:
            raise InvalidAuthorizationTokenException("invalid service assertion header")
        if set(claims) != {"aud", "exp", "authentication"}:
            raise InvalidAuthorizationTokenException("invalid service assertion claims")
        if not isinstance(claims["aud"], str) or claims["aud"] != self.destination:
            raise InvalidAuthorizationTokenException("invalid service assertion audience")

        try:
            verified = jwt.decode(
                assertion,
                self.key,
                algorithms=["HS256"],
                audience=self.destination,
                options={
                    "require": ["aud", "exp", "authentication"],
                    "verify_exp": False,
                },
            )
        except (jwt.PyJWTError, TypeError, ValueError) as error:
            raise InvalidAuthorizationTokenException(
                "invalid service assertion"
            ) from error
        if verified != claims:
            raise InvalidAuthorizationTokenException("invalid service assertion")

        now = int(self.now())
        expires_at = claims["exp"]
        if (
            isinstance(expires_at, bool)
            or not isinstance(expires_at, int)
            or expires_at <= now
            or expires_at > now + MAX_LIFETIME_SECONDS
        ):
            raise InvalidAuthorizationTokenException("invalid service assertion lifetime")

        authentication = claims["authentication"]
        if not isinstance(authentication, dict) or set(authentication) not in (
            {"tenant", "permissions"},
            {"subject", "tenant", "permissions"},
        ):
            raise InvalidAuthorizationTokenException(
                "invalid service assertion authentication"
            )
        tenant_claim = authentication["tenant"]
        if not isinstance(tenant_claim, dict) or set(tenant_claim) != {
            "organization_id",
            "project_id",
        }:
            raise InvalidAuthorizationTokenException("invalid service assertion tenant")
        organization_id = tenant_claim["organization_id"]
        project_id = tenant_claim["project_id"]
        if (
            isinstance(organization_id, bool)
            or isinstance(project_id, bool)
            or not isinstance(organization_id, int)
            or not isinstance(project_id, int)
        ):
            raise InvalidAuthorizationTokenException("invalid service assertion tenant")
        permissions = authentication["permissions"]
        if (
            not isinstance(permissions, list)
            or any(not isinstance(permission, str) for permission in permissions)
            or permissions != sorted(set(permissions))
        ):
            raise InvalidAuthorizationTokenException(
                "invalid service assertion permissions"
            )

        try:
            tenant = Tenant(
                organizationId=organization_id,
                projectId=project_id,
            )
            proof = Proof(
                method="service_assertion",
                audience="rapida_internal",
                reference=ProofReference(
                    type="signing_key", value=self.key_fingerprint
                ),
            )
            asserted_subject = authentication.get("subject")
            if asserted_subject is None and "subject" not in authentication:
                principal = Principal(
                    subject=Identity(
                        type="service",
                        id=INTERNAL_SERVICE_ID,
                        issuer=INTERNAL_SERVICE_ISSUER,
                    ),
                    selectedTenant=tenant,
                    proof=proof,
                    authorities=tuple(permissions),
                )
            else:
                if not isinstance(asserted_subject, dict) or set(
                    asserted_subject
                ) != {"type", "id"}:
                    raise ValueError("invalid asserted subject")
                if (
                    not isinstance(asserted_subject["type"], str)
                    or isinstance(asserted_subject["id"], bool)
                    or not isinstance(asserted_subject["id"], int)
                ):
                    raise ValueError("invalid asserted subject")
                subject = Identity(
                    type=asserted_subject["type"], id=asserted_subject["id"]
                )
                if subject.type == "service":
                    raise ValueError("service subject must not be asserted")
                principal = Principal(
                    subject=subject,
                    actor=ServiceActor(
                        id=INTERNAL_SERVICE_ID,
                        issuer=INTERNAL_SERVICE_ISSUER,
                    ),
                    selectedTenant=tenant,
                    proof=proof,
                    authorities=tuple(permissions),
                )
        except (TypeError, ValueError) as error:
            raise InvalidAuthorizationTokenException(
                "invalid service assertion principal"
            ) from error
        return principal

    @staticmethod
    def decode_closed_json(assertion: str):
        parts = assertion.split(".") if isinstance(assertion, str) else []
        if len(parts) != 3:
            raise InvalidAuthorizationTokenException("invalid service assertion")

        class Pairs(list):
            pass

        def decode(value):
            try:
                source = base64.urlsafe_b64decode(value + "=" * (-len(value) % 4))
                if (
                    base64.urlsafe_b64encode(source).decode("ascii").rstrip("=")
                    != value
                ):
                    raise ValueError("noncanonical base64url")
                parsed = json.loads(source, object_pairs_hook=Pairs)
            except (ValueError, TypeError, json.JSONDecodeError) as error:
                raise InvalidAuthorizationTokenException(
                    "invalid service assertion json"
                ) from error

            def convert(item):
                if isinstance(item, Pairs):
                    result = {}
                    for key, nested in item:
                        if key in result:
                            raise InvalidAuthorizationTokenException(
                                "duplicate service assertion member"
                            )
                        result[key] = convert(nested)
                    return result
                if isinstance(item, list):
                    return [convert(nested) for nested in item]
                return item

            result = convert(parsed)
            if not isinstance(result, dict):
                raise InvalidAuthorizationTokenException(
                    "invalid service assertion json"
                )
            return result

        try:
            signature = base64.urlsafe_b64decode(
                parts[2] + "=" * (-len(parts[2]) % 4)
            )
        except (ValueError, TypeError) as error:
            raise InvalidAuthorizationTokenException(
                "invalid service assertion signature"
            ) from error
        if (
            len(signature) != hashlib.sha256().digest_size
            or base64.urlsafe_b64encode(signature).decode("ascii").rstrip("=")
            != parts[2]
        ):
            raise InvalidAuthorizationTokenException(
                "invalid service assertion signature"
            )
        return decode(parts[0]), decode(parts[1])
