#!/usr/bin/env bash

#----------------------------------------------------------------------
# Functions
#----------------------------------------------------------------------
# Exits the script with the given exit code after waiting
# for a keypress.
#
# @param [Integer] $1 exit code.
function key_exit() {
    echo "Press any key to exit."
    read
    exit $1
}

# Appends a value to an array.
#
# @param [String] $1 Name of the variable to modify
# @param [String] $2 Value to append
function append() {
    eval $1[\${#$1[*]}]=$2
}

#----------------------------------------------------------------------
# Script
#----------------------------------------------------------------------
# Collect the directories and files to remove
my_files=()
append my_files "/opt/vagrant-vmware-desktop"

# Print the files and directories that are to be removed and verify
# with the user that that is what he/she really wants to do.
echo "The following files and directories will be removed:"
for file in "${my_files[@]}"; do
    echo "    $file"
done

echo ""
echo "Do you wish to uninstall Vagrant VMware Utility (Yes/No)?"
read my_answer
if [ "$my_answer" != "Yes" ]; then
    echo "Aborting install. (answer: ${my_answer})"
    key_exit 2
fi

# Initiate the actual uninstall, which requires admin privileges.
echo "The uninstallation process requires administrative privileges"
echo "because some of the installed files cannot be removed by a"
echo "normal user. You may now be prompted for a password..."
echo ""

# Use AppleScript so we can use a graphical `sudo` prompt.
# This way, people can enter the username they wish to use
# for sudo, and it is more Apple-like.
osascript -e "do shell script \"/opt/vagrant-vmware-desktop/bin/vagrant-vmware-utility service uninstall; /bin/rm -Rf ${my_files[*]}\" with administrator privileges"

# Verify that the uninstall succeeded by checking whether every file
# we meant to remove is actually removed.
for file in "${my_files[@]}"; do
    if [ -e "${file}" ]; then
        echo "An error must have occurred since a file that was supposed to be"
        echo "removed still exists: ${file}"
        echo ""
        echo "Please try again."
        key_exit 1
    fi
done

echo "Successfully uninstalled Vagrant VMware Utility."
echo "Done."
key_exit 0
