"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import base64
import json
from pathlib import Path
from unittest.mock import mock_open, patch

import pytest
import yaml

from app.authentication import (
    Identity,
    ServiceAssertionIssuer,
    ServiceAssertionVerifier,
    Tenant,
)
from app.config import get_settings
from app.exceptions.authentication_exception import InvalidAuthorizationTokenException

FIXTURE = json.loads(
    (Path(__file__).parents[1] / "fixtures" / "service_assertion.json").read_text()
)
MALFORMED = json.loads(
    (
        Path(__file__).parents[1]
        / "fixtures"
        / "service_assertion_malformed.json"
    ).read_text()
)
LEGACY_EDDSA_ASSERTION = (
    "eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9."
    "eyJhdWQiOiJkb2N1bWVudC1hcGkiLCJleHAiOjE3OTQ1MjgwNTUsImF1dGhlbnRpY2F0aW9u"
    "Ijp7InRlbmFudCI6eyJvcmdhbml6YXRpb25faWQiOjcsInByb2plY3RfaWQiOjExfSwicGVybW"
    "lzc2lvbnMiOlsib3JnYW5pemF0aW9uLm1lbWJlcnMucmVhZCIsInByb2plY3Qua25vd2xlZGdl"
    "LnJlYWQiXSwic3ViamVjdCI6eyJ0eXBlIjoidXNlciIsImlkIjo0Mn19fQ."
    "cAfyR7mqUMALhZHTI9OWwnrbETBX2uc0AwxZelF0UP4Wfq8PMp49UdywQnmAM8NuIbhY4xKIv6"
    "LRm2QxJeR-Ag"
)


def contract(issuer_now, verifier_now=None):
    issuer = ServiceAssertionIssuer(
        destination=FIXTURE["destination"],
        secret=FIXTURE["secret"],
        now=lambda: issuer_now,
    )
    verifier = ServiceAssertionVerifier(
        destination=FIXTURE["destination"],
        secret=FIXTURE["secret"],
        now=lambda: issuer_now if verifier_now is None else verifier_now,
    )
    return issuer, verifier


def decode_segment(value):
    return json.loads(base64.urlsafe_b64decode(value + "=" * (-len(value) % 4)))


def test_verifies_shared_python_go_golden_assertion():
    _, verifier = contract(FIXTURE["issued_at"])

    principal = verifier.verify(FIXTURE["assertion"])

    assert principal.subject == Identity(type="user", id=42)
    assert principal.actor.id == 1
    assert principal.actor.issuer == "internal"
    assert principal.proof.reference.value == FIXTURE["fingerprint"]


def test_delegated_assertion_uses_generic_internal_actor():
    issuer, verifier = contract(FIXTURE["issued_at"])

    principal = verifier.verify(
        issuer.issue(
            subject=Identity(type="user", id=42),
            tenant=Tenant(organizationId=7, projectId=11),
            permissions=("project.knowledge.read", "organization.members.read"),
        )
    )

    assert principal.subject == Identity(type="user", id=42)
    assert principal.actor.id == 1
    assert principal.actor.issuer == "internal"
    assert principal.selected_tenant == Tenant(organizationId=7, projectId=11)
    assert principal.authorities == (
        "organization.members.read",
        "project.knowledge.read",
    )


@pytest.mark.parametrize("assertion", MALFORMED.values(), ids=MALFORMED.keys())
def test_rejects_shared_malformed_assertions(assertion):
    _, verifier = contract(FIXTURE["issued_at"])

    with pytest.raises(InvalidAuthorizationTokenException):
        verifier.verify(assertion)


def test_direct_assertion_preserves_tenant_and_uses_fingerprint():
    issuer, verifier = contract(FIXTURE["issued_at"])

    principal = verifier.verify(
        issuer.issue(Tenant(organizationId=7, projectId=11))
    )

    assert principal.subject == Identity(type="service", id=1, issuer="internal")
    assert principal.actor is None
    assert principal.selected_tenant == Tenant(organizationId=7, projectId=11)
    assert principal.proof.reference.value == FIXTURE["fingerprint"]


def test_issuer_emits_only_the_accepted_header_and_claims():
    issuer, _ = contract(FIXTURE["issued_at"])

    assertion = issuer.issue(Tenant(organizationId=0, projectId=0))
    header, claims = [decode_segment(part) for part in assertion.split(".")[:2]]

    assert header == {"alg": "HS256", "typ": "JWT"}
    assert set(claims) == {"aud", "exp", "authentication"}
    assert claims["exp"] == FIXTURE["issued_at"] + 55
    assert claims["authentication"] == {
        "tenant": {"organization_id": 0, "project_id": 0},
        "permissions": [],
    }


def test_assertion_from_a_different_shared_secret_is_rejected():
    _, verifier = contract(FIXTURE["issued_at"])
    second_secret = base64.b64encode(bytes(reversed(range(32)))).decode("ascii")
    second_issuer = ServiceAssertionIssuer(
        destination=FIXTURE["destination"],
        secret=second_secret,
        now=lambda: FIXTURE["issued_at"],
    )

    with pytest.raises(InvalidAuthorizationTokenException):
        verifier.verify(
            second_issuer.issue(Tenant(organizationId=0, projectId=0))
        )


@pytest.mark.parametrize(
    "relative_path",
    ("env/document.yaml", "docker/document-api/config.yaml"),
)
def test_document_configs_use_the_shared_secret(relative_path):
    repository = Path(__file__).parents[4]
    configured = yaml.safe_load((repository / relative_path).read_text())

    assert configured["secret"] == FIXTURE["secret"]
    assert "service_assertion" not in configured


@pytest.mark.parametrize("issuer_offset", [-5, 5])
def test_clock_skew_within_budget_is_accepted(issuer_offset):
    now = FIXTURE["issued_at"]
    issuer, verifier = contract(now + issuer_offset, verifier_now=now)

    assert verifier.verify(
        issuer.issue(Tenant(organizationId=0, projectId=0))
    ).subject.id == 1


def test_clock_skew_beyond_budget_is_rejected():
    now = FIXTURE["issued_at"]
    issuer, verifier = contract(now + 6, verifier_now=now)

    with pytest.raises(InvalidAuthorizationTokenException):
        verifier.verify(issuer.issue(Tenant(organizationId=0, projectId=0)))


def test_expired_wrong_audience_and_legacy_assertions_are_rejected():
    issuer, verifier = contract(
        FIXTURE["issued_at"], verifier_now=FIXTURE["issued_at"] + 55
    )
    with pytest.raises(InvalidAuthorizationTokenException):
        verifier.verify(issuer.issue(Tenant(organizationId=0, projectId=0)))

    _, wrong_audience_verifier = contract(FIXTURE["issued_at"])
    wrong_audience = ServiceAssertionIssuer(
        "web-api",
        FIXTURE["secret"],
        now=lambda: FIXTURE["issued_at"],
    ).issue(Tenant(organizationId=0, projectId=0))
    with pytest.raises(InvalidAuthorizationTokenException):
        wrong_audience_verifier.verify(wrong_audience)

    with pytest.raises(InvalidAuthorizationTokenException):
        wrong_audience_verifier.verify(LEGACY_EDDSA_ASSERTION)


@pytest.mark.parametrize(
    "secret",
    (
        "",
        "not-base64",
        base64.b64encode(bytes(31)).decode("ascii"),
        "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh9=",
        "é" * 32,
    ),
)
def test_invalid_shared_secret_fails_configuration(secret):
    repository = Path(__file__).parents[4]
    source = (repository / "env/document.yaml").read_text()
    source = source.replace(
        f'secret: "{FIXTURE["secret"]}"', f'secret: "{secret}"', 1
    )
    get_settings.cache_clear()
    try:
        with patch("builtins.open", mock_open(read_data=source)):
            with pytest.raises(
                ValueError, match="application configuration is invalid"
            ):
                get_settings()
    finally:
        get_settings.cache_clear()


@pytest.mark.parametrize(
    "legacy",
    ("service_assertion: null", "service_assertion:\n  key_file: legacy.key"),
)
def test_legacy_service_assertion_configuration_is_rejected(legacy):
    repository = Path(__file__).parents[4]
    source = (repository / "env/document.yaml").read_text()
    source += "\n" + legacy + "\n"
    get_settings.cache_clear()
    try:
        with patch("builtins.open", mock_open(read_data=source)):
            with pytest.raises(
                ValueError,
                match="legacy service assertion configuration is not supported",
            ):
                get_settings()
    finally:
        get_settings.cache_clear()


def test_malformed_authentication_secret_is_not_exposed():
    repository = Path(__file__).parents[4]
    source = (repository / "env/document.yaml").read_text()
    secret = FIXTURE["secret"]
    source = source.replace(
        f'secret: "{secret}"',
        f'secret:\n  - "{secret}"',
        1,
    )
    get_settings.cache_clear()
    try:
        with patch("builtins.open", mock_open(read_data=source)):
            with pytest.raises(ValueError) as raised:
                get_settings()
        assert str(raised.value) == "application configuration is invalid"
        assert secret not in str(raised.value)
        assert raised.value.__cause__ is None
    finally:
        get_settings.cache_clear()


def test_malformed_yaml_does_not_expose_authentication_secret():
    secret = FIXTURE["secret"]
    source = f'secret: "{secret}\\q"\n'
    get_settings.cache_clear()
    try:
        with patch("builtins.open", mock_open(read_data=source)):
            with pytest.raises(ValueError) as raised:
                get_settings()
        assert str(raised.value) == "application configuration is invalid"
        assert secret not in str(raised.value)
        assert raised.value.__cause__ is None
    finally:
        get_settings.cache_clear()
