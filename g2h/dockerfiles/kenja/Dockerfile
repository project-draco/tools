FROM ubuntu:latest

RUN apt-get update && apt-get install -y --no-install-recommends \
        default-jdk \
        git \
        patch \
        python \
        python-dev \
        python-distribute \
        python-pip \
        ssh \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /usr/local/share/ca-certificates \
    && cd /

COPY badname.patch /
COPY badname2.patch /
COPY missing-email.patch /
COPY unicode.patch /
COPY peeled.patch /
COPY converter.sh /usr/local/bin
COPY certificates/* /usr/local/share/ca-certificates

RUN update-ca-certificates \
    && git clone https://github.com/project-draco/kenja.git \
    && cd /kenja \
    && git checkout -q reduce-memory-usage \
    && pip install setuptools \
    && echo 'y\ny\n' | python setup.py install \
    && sed -i '10s/GitCommandError/GitCommandError,/' /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/repo/base.py \
    && sed -i '10a\    BadName' /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/repo/base.py \
    && patch /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/repo/base.py /badname.patch \
    && patch /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/objects/commit.py /badname2.patch \
    && patch /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/objects/commit.py /missing-email.patch \
    && patch /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/refs/symbolic.py /unicode.patch \
    && patch /usr/local/lib/python2.7/dist-packages/GitPython-0.3.6-py2.7.egg/git/refs/symbolic.py /peeled.patch \
    && pip install wheel \
    && pip install gitdb-speedups
