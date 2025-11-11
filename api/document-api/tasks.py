"""
Copyright (c) 2024 Prashant Srivastav <prashant@rapida.ai>
All rights reserved.

This code is licensed under the MIT License. You may obtain a copy of the License at
https://opensource.org/licenses/MIT.

Unless required by applicable law or agreed to in writing, software distributed under the
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
either express or implied. See the License for the specific language governing permissions
and limitations under the License.

"""
from typing import Any, Dict

import invoke

from app.config import get_settings
from app.configs.postgres_config import PostgresConfig
from app.configs.redis_config import RedisConfig


@invoke.task
def lint(ctx):
    """Run linter."""
    ctx.run("pre-commit run --all-files --show-diff-on-failure")


@invoke.task
def dev(ctx):
    """Run the app with config."""
    setting = get_settings()
    ctx.run(
        " ".join(
            [
                "sh",
                "./bin/start.sh",
                "--reload",
                "--log-level=debug",
                f"--host={setting.host}",
                f"--port={setting.port}",
            ]
        )
    )


@invoke.task
def requirements(ctx):
    """Generate requirements.txt"""
    reqs = [
        "pipenv requirements > requirements.txt",
        "pipenv requirements --dev-only > requirements-dev.txt",
    ]
    [ctx.run(req) for req in reqs]


@invoke.task
def test(ctx):
    """Run pytest tests."""
    ctx.run(
        " ".join(
            [
                "pytest",
                "-v",
                "-s",
                "--cov-report term",
                "--cov-report xml",
                "--cov=app",
                "--asyncio-mode=auto",
            ]
        )
    )


@invoke.task
def git(ctx):
    """Run to set up git hooks"""
    ctx.run("chmod +x bin/git-commit-hook-setup.sh")
    ctx.run("sh bin/git-commit-hook-setup.sh")
    # pre-commit installed hook for lint
    ctx.run("pre-commit install")


@invoke.task
def setup(ctx):
    # setup git hooks for commit
    git(ctx)
    requirements(ctx)


def servicize(name: str, port: int, component: Dict, docker_file_path: str) -> Dict:
    """Docker component formations"""
    component["hostname"] = f"{name}"
    component["container_name"] = f"{name}"
    component["build"] = {"context": "./", "dockerfile": docker_file_path}
    component["restart"] = "always"
    component["ports"] = [f"{str(port)}:{str(port)}"]
    component["networks"] = {"lomotif": {"aliases": [f"{name}"]}}
    return component


@invoke.task
def dockerize(ctx):
    """Generate docker-composer file"""
    import ruamel.yaml as yaml

    setting = get_settings()

    def postgres_docker_definition(name: str, config: PostgresConfig) -> Dict:
        postgres_service = {
            "environment": [
                f"POSTGRES_DATABASE={config.db}",
                f"POSTGRES_USER={config.auth.user}",
                f"POSTGRES_PASSWORD={config.auth.password.get_secret_value()}",
            ]
        }
        return servicize(
            name=name,
            component=postgres_service,
            docker_file_path="dockerfiles/postgres.dockerfile",
            port=config.port,
        )

    def redis_docker_definition(name: str, config: RedisConfig) -> Dict:
        return servicize(
            name=name,
            component={},
            docker_file_path="dockerfiles/redis.dockerfile",
            port=config.port,
        )

    composition: Dict[str, Any] = {
        "version": "3.1",
        "services": {},
        "networks": {"lomotif": {"name": "lomotif_main_network"}},
    }

    service_list = []
    for key in setting.dict():
        if type(getattr(setting, key)) is RedisConfig:
            composition["services"][key] = redis_docker_definition(
                key, getattr(setting, key)
            )
            service_list.append(key)

        if type(getattr(setting, key)) is PostgresConfig:
            composition["services"][key] = postgres_docker_definition(
                key, getattr(setting, key)
            )
            service_list.append(key)

    main_app = {
        "env_file": [".env"],
        "command": " ".join(
            ["sh", "./bin/start.sh", f"--host={setting.host}", f"--port={setting.port}"]
        ),
    }

    composition["services"]["app"] = servicize(
        name="app",
        component=main_app,
        docker_file_path="Dockerfile",
        port=setting.port,
    )

    if len(service_list) > 0:
        composition["services"]["app"]["depends_on"] = service_list

    with open("docker-composer.yml", "w") as outfile:
        yaml.dump(
            composition,
            stream=outfile,
            Dumper=yaml.SafeDumper,
            indent=4,
            default_flow_style=False,
        )
