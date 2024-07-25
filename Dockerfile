FROM alpine/helm:3.5.3 as helm
FROM alpine:3

# Install apps on $PATH
COPY --from=helm /usr/bin/helm /usr/bin/helm
COPY bin/mirror /usr/bin/helm-mirror

# # Install as a Helm plugin
COPY  LICENSE /root/.local/share/helm/plugins/mirror/LICENSE
COPY  README.md /root/.local/share/helm/plugins/mirror/README.md
COPY  bin/mirror /root/.local/share/helm/plugins/mirror/bin/mirror
COPY  plugin.yaml /root/.local/share/helm/plugins/mirror/plugin.yaml
COPY  scripts/install-binary.sh /root/.local/share/helm/plugins/mirror/scripts/install-binary.sh
