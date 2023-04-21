#!/bin/zsh

# use go-licenses for Go dependency scanning
export GO111MODULE="on"

git clone git@github.com:google/go-licenses.git

cd go-licenses

go mod download

go install github.com/google/go-licenses@latest

cd ../functions/source/soci-index-generator-lambda

# license scanning
modules=$(go-licenses report github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda)

PROJECT_MODULE="github.com/aws-ia/cfn-aws-soci-index-builder/soci-index-generator-lambda"

# use pip-licenses with pipreqs for Python dependency scanning
pip install -U pip-licenses

# install pipreqs that generates requirements.txt which incude all Python packages the project uses
pip install -U pipreqs

# Generate requirements.txt
cd ../../../ && pipreqs .

# Print scanning results
echo "+===========================================================+"
echo "                      Go Dependencies"
echo "+===========================================================+"
echo "|            Package                          License"
echo "+-------------------------------------+---------------------+"

while IFS=',' read -r package _ license; do
    # skip project modules
    if [[ "$package" == "$PROJECT_MODULE"* ]]; then
        continue
    fi
        
    printf "| % 30s  %20s \n" $package $license
    echo "+-------------------------------------+---------------------+"
done <<< "$modules"

echo "+====================================+"
echo "         Python Dependencies"
echo "+====================================+"
while read line
do
  # extract package names
  packages=$(echo $line | cut -d "=" -f 1)
  
  # license scanning
  pip-licenses -p $packages --format=rst
done < requirements.txt

# clean up
rm go-licenses -f -r
rm requirements.txt