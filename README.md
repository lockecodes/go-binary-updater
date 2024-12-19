# Go Binary Updater
This library aims to make it easy to upgrade a cli binary as a command from said binary.

#### What does this library do?
1. Query the repository for all releases for a project
2. Filter the releases down to the latest version
3. Download the latest release binary `tar.gz` file
4. Extract the file to a temporary location
5. Move the file to a "versioned" directory
6. Symlink to the latest binary in that versioned directory
