set dotenv-load
set shell := ["bash", "-euo", "pipefail", "-c"]

# Show all available recipes.
default:
    @just --list

alias help := default

import 'just/docker.just'
import 'just/images.just'
import 'just/run.just'
import 'just/development.just'
import 'just/ci.just'
