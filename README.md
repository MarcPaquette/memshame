# Memshame
Hall of Shame:  A cf-cli plugin that can be used to find applications instance (AI) discrepancies between the amount of memory allocated to the AI and the amount of memory used by the AI.  Helpful for finding waste within a foundation

# Build Instructions
clone this respository && cd into it\n
run "go mod init memshame"\n
run "go mod tidy"\n
run "go build"\n

# Installation Instructions
cf install-plugin <path to build above>/memshame\n

# Usage Instructions
cf memshame -h :  displays help information\n
cf memshame      :  will run the plugin against the entire foundation\n
cf memshame -o <org_name>  :  Runs plugin against all applications within an org\n
cf memshame -o <org_name> -s <space_name>  :  Runs plugin against all applications within an org & space combo\n
cf memshame -hr :  Outputs plugin memory information in megabytes for readability.  Note this flag can be used with any of the above commands. \n


