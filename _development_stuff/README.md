# Development

Spins up a local Traefik instance and a dummy `whoami` service so the *traefik* plugin has real
routers to discover.

1. Build CoreDNS with this plugin (`go generate && go build`, or `make`, from the CoreDNS root).
2. Start Traefik + whoami:

   ~~~ sh
   docker compose up
   ~~~

   Traefik's API is now on `http://localhost:8080/api` and its dashboard on
   `http://localhost:8080/dashboard/`.
3. Run CoreDNS with the Corefile in this directory:

   ~~~ sh
   coredns -conf Corefile
   ~~~

   Port `5354` (not `53`) is used so CoreDNS needs no root and avoids the mDNS/Bonjour listener on
   macOS. Any free high port works.
4. Query the host discovered from Traefik. Within `refreshinterval` seconds it resolves:

   ~~~ sh
   dig @127.0.0.1 -p 5354 whoami.example.org A +short   # -> 127.0.0.1
   ~~~

Add more services to `docker-compose.yml` with a ``traefik.http.routers.<name>.rule=Host(`...`)``
label and they show up in CoreDNS on the next poll.
