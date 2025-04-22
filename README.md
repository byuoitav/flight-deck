# Flight-Deck

Flight deck is an API to trigger ansible deployments.

Champagne Code is setup as follows: 

Champagne - Pi (client) -> tar.gz file that contains the binary of the go code along with the template and public folders that contains an Angular web site that used as a frontend to the deployment process. The Go code gets compiled and then tar balled with the template and public folders and is pulled down as part of the deploy process directly on the Pi.

Champagne - Server ->  This is an intermediary that allows for the the client running on the Pi to make calls to Tyk (API Gateway) which will then run an Ansible playbook

CMD/flightdeck - This is the main service that runs on the Ansible server that translates the calls from the Champagne server to Ansible and executes ansible playbooks
Most of the code in the repository deals with this service.

Flightdeck files - The scripts and files that are pulled down from the image scripts that actually run the Champagne Pi Client on a Pi.  

Image - files installed directly in the Pi image on build that will reach out to github and download the flightdeck files. This runs as a service on the Pi when it boots and will be disabled by the Ansible deployment playbook. 

Currently, we only use the binary directly installed on each system.  We could use docker down the road to run these service.  
