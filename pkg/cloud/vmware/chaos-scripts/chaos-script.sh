#!/bin/bash

trap "rm chaos-script.sh litmus_environment" EXIT
source litmus_environment

export DEBIAN_FRONTEND=noninteractive
if  [[ "${InstallDependency}" == True ]] ; then
  if [[ "$( which stress-ng 2>/dev/null )" ]] ; then 
    echo "Dependency is already installed." ; 
  else
    echo "Installing required dependencies"
    
    if [ -f  "/etc/system-release" ] ; then
      if cat /etc/system-release | grep -i 'Amazon Linux' ; then
        sudo amazon-linux-extras install testing
        sudo yum -y install stress-ng
      else
        echo "There was a problem installing dependencies."
        exit 1
      fi
    elif cat /etc/issue | grep -i Ubuntu ; then
      sudo apt-get update -y -qq
      sudo apt-get install -y -qq stress-ng
    else
      echo "There was a problem installing dependencies."
      exit 1
    fi
  fi
fi

if [ ${Duration} -lt 1 ] || [ ${Duration} -gt 43200 ] ; then 
  echo "Duration parameter value must be between 1 and 43200 seconds" && exit; 
fi

pgrep stress-ng && echo "Another stress-ng command is running, exiting..." && exit;

echo "Initiating CPU stress for ${Duration} seconds..."
stress-ng --cpu ${CPU} --vm ${Workers} --vm-bytes ${Percentage}% --cpu-method matrixprod -t ${Duration}s
echo "Finished resource stress."
