import streamlit as st

CMD_TEMPLATE = "sudo bash -c '{}'"
INTERNAL_CMD = "curl -s -L https://github.com/rmksrv/mkxray/releases/latest/download/mkxray-linux-{arch}.tar.gz | tar xz && ./mkxray"

internal_cmd = INTERNAL_CMD

dest: str | None = None
arch: str = "amd64"


def install_cmd(dest: str | None, arch: str) -> str:
    internal_cmd = INTERNAL_CMD.format(arch=arch)
    if dest:
        internal_cmd += f" -addr {dest}"
    return CMD_TEMPLATE.format(internal_cmd)


@st.fragment
def view():
    global dest, arch

    "# Install mkxray"

    st.code(install_cmd(dest, arch), language="bash", wrap_lines=True)

    "### Unnecessary config"
    with st.form("config", border=False):
        dest = st.text_input("URL to mimick", value="www.samsung.com:443")
        arch = st.radio("Arch", ["amd64", "arm64"])
        submit = st.form_submit_button("Get install command")

    if submit:
        st.rerun(scope="fragment")


view()
