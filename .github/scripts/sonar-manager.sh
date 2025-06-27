#!/bin/bash

# Prevent command echoing and exit on any error
set +x -e

# Clear sensitive environment variables on exit
trap 'unset SONAR_TOKEN' EXIT

# sonar-manager.sh
# ---------------
# Manages SonarCloud projects for the docker/go-sdk repository
#
# Usage:
#   ./.github/scripts/sonar-manager.sh -h                            # Show help message
#   ./.github/scripts/sonar-manager.sh -a create -p <project>        # Create a single project
#   ./.github/scripts/sonar-manager.sh -a delete -p <project>        # Delete a single project
#   ./.github/scripts/sonar-manager.sh -a renameBranch -p <project>  # Rename the main branch to 'main'
#   ./.github/scripts/sonar-manager.sh -a createAll                  # Create all projects
#   ./.github/scripts/sonar-manager.sh -a deleteAll                  # Delete all projects
#
# Project name format:
#   - For modules: modules_<module-name>
#
# Examples:
#   ./.github/scripts/sonar-manager.sh -a create -p modules_mysql
#   ./.github/scripts/sonar-manager.sh -a delete -p modules_mysql
#   ./.github/scripts/sonar-manager.sh -a renameBranch -p modules_mysql
#   ./.github/scripts/sonar-manager.sh -a createAll
#   ./.github/scripts/sonar-manager.sh -a deleteAll
#
# Environment variables:
#   SONAR_TOKEN - Required. The SonarCloud authentication token.

SONAR_TOKEN=${SONAR_TOKEN:-}
ORGANIZATION="docker"

# list all modules
MODULES=$(go work edit -json | jq -r '.Use[] | "\(.DiskPath | ltrimstr("./"))"' | tr '\n' ' ' && echo)

# Delete all the projects for the go-sdk in SonarCloud.
deleteAll() {
    for MODULE in $MODULES; do
        delete "$MODULE"
    done
}

# Helper function to print success message
print_success() {
    local action=$1
    local module=$2
    echo -e "\033[32mOK\033[0m: $action $([ ! -z "$module" ] && echo "for $module")"
}

# Helper function to print failure message
print_failure() {
    local action=$1
    local module=$2
    local status=$3
    echo -e "\033[31mFAIL\033[0m: Failed to $action $([ ! -z "$module" ] && echo "for $module") (HTTP status: $status)"
    exit 1
}

# Helper function to handle curl responses
handle_curl_response() {
    local response_code=$1
    local action=$2
    local module=$3
    local allow_404=${4:-false}  # Optional parameter to allow 404 responses

    if [ $response_code -eq 200 ] || [ $response_code -eq 204 ] || ([ "$allow_404" = "true" ] && [ $response_code -eq 404 ]); then
        return
    fi
    print_failure "$action" "$module" "$response_code"
}

# Delete a project in SonarCloud
delete() {
    MODULE=$1
    NAME=$(echo $MODULE | tr '_' '-')
    PROJECT_KEY="docker_go-sdk_${MODULE}"
    PROJECT_NAME="go-sdk/${NAME}"
    response=$(curl -s -w "%{http_code}" -X POST https://${SONAR_TOKEN}@sonarcloud.io/api/projects/delete \
        -d "name=${PROJECT_NAME}&project=${PROJECT_KEY}&organization=docker" 2>/dev/null)
    status_code=${response: -3}
    handle_curl_response $status_code "delete" "$MODULE"
    print_success "Deleted project" "$MODULE"
}

# Create all the projects for the go-sdk in SonarCloud.
createAll() {
    for MODULE in $MODULES; do
        create "$MODULE"
    done
}

# create a new project in SonarCloud
create() {
    MODULE=$1
    NAME=$(echo $MODULE | tr '_' '-')
    PROJECT_KEY="docker_go-sdk_${MODULE}"
    PROJECT_NAME="go-sdk/${NAME}"
    response=$(curl -s -w "%{http_code}" -X POST https://${SONAR_TOKEN}@sonarcloud.io/api/projects/create \
        -d "name=${PROJECT_NAME}&project=${PROJECT_KEY}&organization=docker" 2>/dev/null)
    status_code=${response: -3}
    handle_curl_response $status_code "create" "$MODULE"
    print_success "Created project" "$MODULE"
}

# Rename all the main branches to the new name for all the projects for the go-sdk in SonarCloud.
renameAllMainBranches() {
    for MODULE in $MODULES; do
        rename_main_branch "$MODULE"
    done
}

# rename the main branch to the new name: they originally have the name "master"
rename_main_branch() {
    MODULE=$1
    NAME=$(echo $MODULE | tr '_' '-')
    PROJECT_KEY="docker_go-sdk_${MODULE}"
    PROJECT_NAME="go-sdk/${NAME}"
    
    # Delete main branch (404 is acceptable here)
    response=$(curl -s -w "%{http_code}" -X POST https://${SONAR_TOKEN}@sonarcloud.io/api/project_branches/delete \
        -d "branch=main&project=${PROJECT_KEY}&organization=docker" 2>/dev/null)
    status_code=${response: -3}
    handle_curl_response $status_code "delete main branch" "$MODULE" true
    
    # Rename master to main
    response=$(curl -s -w "%{http_code}" -X POST https://${SONAR_TOKEN}@sonarcloud.io/api/project_branches/rename \
        -d "name=main&project=${PROJECT_KEY}&organization=docker" 2>/dev/null)
    status_code=${response: -3}
    handle_curl_response $status_code "rename branch to main" "$MODULE"
    print_success "Renamed branch to main" "$PROJECT_KEY"
}

show_help() {
    echo "Usage: $0 [-h] [-a ACTION] [-p PROJECT_NAME]"
    echo
    echo "Options:"
    echo "  -h            Show this help message"
    echo "  -a ACTION     Action to perform (create, delete, renameBranch, createAll, deleteAll)"
    echo "  -p PROJECT    Project name to operate on (required for create/delete/renameBranch)"
    echo
    echo "Actions:"
    echo "  create        Creates a new SonarCloud project and sets up the main branch"
    echo "  delete        Deletes an existing SonarCloud project"
    echo "  renameBranch  Renames the main branch to 'main' (default is 'master')"
    echo "  createAll     Creates all projects and sets up their main branches"
    echo "  deleteAll     Deletes all existing projects"
    echo
    echo "Examples:"
    echo "  $0 -a create -p modules_mymodule        # Create a new project"
    echo "  $0 -a delete -p modules_mymodule        # Delete an existing project"
    echo "  $0 -a renameBranch -p modules_mymodule  # Rename the main branch to 'main'"
    echo "  $0 -a createAll                         # Create all projects"
    echo "  $0 -a deleteAll                         # Delete all projects"
}

validate_action() {
    local action=$1
    case $action in
        create|delete|renameBranch|createAll|deleteAll)
            return 0
            ;;
        *)
            echo "Error: Invalid action '$action'. Valid actions are: create, delete, renameBranch, createAll, deleteAll"
            return 1
            ;;
    esac
}

validate_project() {
    local action=$1
    local project=$2
    
    # Skip project validation for "All" actions
    if [[ "$action" == *"All" ]]; then
        return 0
    fi
    
    if [ -z "$project" ]; then
        echo "Error: Project name is required for action '$action'. Use -p to specify a project name"
        return 1
    fi

    # Check if the project exists in the MODULES list
    if ! echo " $MODULES " | grep -q " $project "; then
        echo "Error: Project '$project' not found in the MODULES list: $MODULES"
        return 1
    fi

    return 0
}

main() {
    local project_name=""
    local action=""

    # Handle flags
    while getopts "ha:p:" opt; do
        case $opt in
            h)
                show_help
                exit 0
                ;;
            a)
                action="$OPTARG"
                if ! validate_action "$action"; then
                    exit 1
                fi
                ;;
            p)
                project_name="$OPTARG"
                ;;
            \?)
                echo "Invalid option: -$OPTARG" >&2
                show_help
                exit 1
                ;;
        esac
    done

    # Validate SONAR_TOKEN is set (except for help)
    if [ -z "${SONAR_TOKEN}" ]; then
        echo "Error: SONAR_TOKEN environment variable is not set"
        exit 1
    fi

    # Validate project name is provided for non-All actions
    if ! validate_project "$action" "$project_name"; then
        exit 1
    fi

    case $action in
        create)
            create "$project_name"
            rename_main_branch "$project_name"
            ;;
        delete)
            delete "$project_name"
            ;;
        renameBranch)
            rename_main_branch "$project_name"
            ;;
        createAll)
            createAll
            renameAllMainBranches
            ;;
        deleteAll)
            deleteAll
            ;;
        *)
            echo "No valid action specified. Use -h for help"
            ;;
    esac
}

main "$@"
