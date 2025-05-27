FROM mcr.microsoft.com/vscode/devcontainers/base:ubuntu-22.04

SHELL ["/bin/bash", "-lc"]

RUN apt-get update


USER vscode


RUN git clone https://github.com/asdf-vm/asdf.git ~/.asdf --branch v0.14.0 && \
    echo "source $HOME/.asdf/asdf.sh" >> ~/.bashrc && \
    echo "source $HOME/.asdf/completions/asdf.bash" >> ~/.bashrc


RUN /bin/bash -c "source $HOME/.asdf/asdf.sh && \
    asdf plugin add golang && \
    asdf install golang 1.23.4 && \
    asdf global golang 1.23.4"

    
RUN /bin/bash -c "source $HOME/.asdf/asdf.sh && \
    asdf plugin add opentofu && \
    asdf install opentofu 1.8.5 && \
    asdf global opentofu 1.8.5"


# 1.5.X are the last open source versions of Terraform (using Mozilla Public License, version 2.0)
RUN /bin/bash -c "source $HOME/.asdf/asdf.sh && \
    asdf plugin add terraform && \
    asdf install terraform 1.5.7 && \
    asdf global terraform 1.5.7"


RUN /bin/bash -c "source $HOME/.asdf/asdf.sh && \
    go install -v github.com/go-delve/delve/cmd/dlv@v1.24.0"


ENTRYPOINT [ "/usr/local/share/docker-init.sh" ]
CMD [ "sleep", "infinity" ]