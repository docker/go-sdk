# Contributing

`Docker SDK for Go` is open source, and we love to receive contributions from our community — you!

There are many ways to contribute, from writing tutorials or blog posts, improving the documentation, submitting bug reports and feature requests, or writing code for the any of the modules in the repository.

In any case, if you like the project, please star the project on [GitHub](https://github.com/docker/go-sdk/stargazers) and help spread the word :)
Also join our [Docker Community Slack workspace](https://communityinviter.com/apps/dockercommunity/docker-community) to get help, share your ideas, and chat with the community.

## Questions

GitHub is reserved for bug reports and feature requests; it is not the place for general questions.
If you have a question or an unconfirmed bug, please visit our [Docker Community Slack workspace](https://communityinviter.com/apps/dockercommunity/docker-community);
feedback and ideas are always welcome.

## Code contributions

If you have a bug fix or new feature that you would like to contribute, please find or open an [issue](https://github.com/docker/go-sdk/issues) first.
It's important to talk about what you would like to do, as there may already be someone working on it,
or there may be context to be aware of before implementing the change.

Next would be to **fork** the repository and make your changes in a feature branch. **Please do not commit changes to the `main` branch**,
otherwise we won't be able to contribute to your changes directly in the PR.

### Submitting your changes

Please just be sure to:

* follow the style, naming and structure conventions of the rest of the project.
* make commits atomic and easy to merge.
* use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) for the PR title. This will help us to understand the nature of the changes, and to generate the changelog after all the commits in the PR are squashed.
    * Please use the `feat!`, `chore!`, `fix!`... types for breaking changes, as these categories are considered as `breaking change` in the changelog. Please use the `!` to denote a breaking change.
    * Please use the `security` type for security fixes, as these categories are considered as `security` in the changelog.
    * Please use the `feat` type for new features, as these categories are considered as `feature` in the changelog.
    * Please use the `fix` type for bug fixes, as these categories are considered as `bug` in the changelog.
    * Please use the `docs` type for documentation updates, as these categories are considered as `documentation` in the changelog.
    * Please use the `chore` type for housekeeping commits, including `build`, `ci`, `style`, `refactor`, `test`, `perf` and so on, as these categories are considered as `chore` in the changelog.
    * Please use the `deps` type for dependency updates, as these categories are considered as `dependencies` in the changelog.

> ⚠️ **Important**
>
> There is a GitHub Actions workflow that will check if your PR title follows the conventional commits convention. If not, it contributes a failed check to your PR.
> To know more about the conventions, please refer to the [workflow file](./.github/workflows/conventions.yml).

* use [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) for your commit messages, as it improves the readability of the commit history, and the review process. Please follow the above conventions for the PR title.
* unless necessary, please try to **avoid pushing --force** to the published branch you submitted a PR from, as it makes it harder to review the changes from a given previous state.
* apply format running `make lint` for the module you are contributing to. It will run `golangci-lint` for the module you are contributing to with the configuration set in the root directory of the project. Please be aware that the lint stage on CI could fail if this is not done.
* verify all tests for the module you are contributing to are passing. Build and test the project with `make test` to do this.
* when updating the `go.mod` file, please run `go work sync` to ensure all modules are updated.
