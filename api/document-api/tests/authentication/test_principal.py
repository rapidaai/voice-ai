"""
Copyright (c) 2023-2025 RapidaAI
Author: Prashant Srivastav <prashant@rapida.ai>

Licensed under GPL-2.0 with Rapida Additional Terms.
See LICENSE.md for details or contact sales@rapida.ai for commercial use.
"""

import pytest

from app.authentication import Authentication


def user_authentication(organization_id=9, project_id=11):
    return {
        "principal": {
            "subject": {"type": "user", "id": 7},
            "selectedTenant": {
                "organizationId": organization_id,
                "projectId": project_id,
            },
            "proof": {
                "method": "user_session",
                "reference": {"type": "user_session", "value": "8"},
            },
        },
        "user": {
            "id": 7,
            "name": "Ada",
            "email": "ada@example.com",
            "status": "ACTIVE",
        },
        "organizations": [],
    }


def test_selected_role_graph_derives_one_authority_snapshot():
    value = user_authentication()
    value["organizations"] = [
        {
            "id": 9,
            "name": "Rapida",
            "roles": [
                {
                    "id": 1,
                    "key": "organization-member",
                    "name": "Member",
                    "scope": "ORGANIZATION",
                    "subject": "USER",
                    "definitionType": "SYSTEM",
                    "permissions": [
                        {
                            "id": 1,
                            "key": "organization.members.read",
                            "scope": "ORGANIZATION",
                            "name": "Read members",
                        }
                    ],
                }
            ],
            "projects": [
                {
                    "id": 11,
                    "name": "Voice",
                    "roles": [
                        {
                            "id": 2,
                            "key": "project-viewer",
                            "name": "Viewer",
                            "scope": "PROJECT",
                            "subject": "USER",
                            "definitionType": "SYSTEM",
                            "permissions": [
                                {
                                    "id": 2,
                                    "key": "project.knowledge.read",
                                    "scope": "PROJECT",
                                    "name": "Read knowledge",
                                }
                            ],
                        }
                    ],
                }
            ],
        }
    ]

    authentication = Authentication.model_validate(value)

    assert authentication.principal.authorities == (
        "organization.members.read",
        "project.knowledge.read",
    )


def test_tenantless_onboarding_user_is_valid_without_roles():
    authentication = Authentication.model_validate(user_authentication(0, 0))

    assert authentication.principal.authorities == ()


def test_tenantless_multi_organization_user_keeps_role_graph_without_authorities():
    value = user_authentication(0, 0)
    role = {
        "id": 1,
        "key": "organization-member",
        "name": "Member",
        "scope": "ORGANIZATION",
        "subject": "USER",
        "definitionType": "SYSTEM",
        "permissions": [
            {
                "id": 1,
                "key": "organization.members.read",
                "scope": "ORGANIZATION",
                "name": "Read members",
            }
        ],
    }
    value["organizations"] = [
        {"id": 9, "name": "Rapida", "roles": [role], "projects": []},
        {"id": 10, "name": "Acme", "roles": [role], "projects": []},
    ]

    authentication = Authentication.model_validate(value)

    assert authentication.principal.selected_tenant.organization_id == 0
    assert authentication.principal.authorities == ()
    assert len(authentication.organizations) == 2


def test_selected_tenant_must_exist_in_aggregate():
    with pytest.raises(ValueError, match="selected organization"):
        Authentication.model_validate(user_authentication())
