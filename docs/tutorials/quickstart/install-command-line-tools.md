# Install your own command-line tools

Now that you've finished [Launch your first workspace](./launch-workspace.md),
you can add your favorite command-line tools to every workspace.

The Quickstart template installs system languages through the **Programming Languages** parameter,
but it doesn't carry the small command-line tools you may often use,
such as [`bat`](https://github.com/sharkdp/bat) or [`ripgrep`](https://github.com/BurntSushi/ripgrep).
You can install those yourself with a package manager like [Homebrew](https://brew.sh/) or [mise](https://mise.jdx.dev/).

In this guide, you install both Homebrew and mise,
install a tool with each,
and learn which installs survive a workspace restart and why.
You then change the template so the Homebrew tools persist too.

> [!NOTE]
> This guide works inside a running workspace from the Quickstart template.
> Most of it runs in the workspace, but the last step edits the template so Homebrew can persist.

## What you'll do

- ✅ Install command-line tools with [Homebrew](https://brew.sh/) and [mise](https://mise.jdx.dev/) into your workspace.
- ✅ Restart the workspace and see which tools persist.
- ✅ Learn why one persists and the other doesn't.
- ✅ Wire up Homebrew so its tools persist too.

## What persists in a workspace

A Quickstart workspace keeps your home directory, `/home/coder`, on a persistent volume.
Everything outside `/home/coder` comes from the workspace image,
and Coder rebuilds it from that image every time the workspace starts.

A tool survives a restart only when both of these are true:

- The tool installs into `/home/coder`.
- Your shell finds the tool through a file in `/home/coder`, such as `.bashrc`.

You'll install tools two ways and restart to see this rule decide which ones stay,
then change the template so Homebrew follows the rule too.

## Step 1: Install Homebrew and mise

Open a terminal in your workspace.

Install [Homebrew](https://brew.sh/) with its setup script:

```sh
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

Homebrew installs to `/home/linuxbrew/.linuxbrew`.
Add it to your shell so the `brew` command is available:

```sh
echo 'eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"' >> ~/.bashrc
eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
```

Install [mise](https://mise.jdx.dev/) with its setup script:

```sh
curl -fsSL https://mise.run | sh
```

mise installs to `~/.local/bin/mise`, inside your home directory.
Activate it so every new shell loads it:

```sh
echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc
```

Activation only takes effect in shells that start after this change,
so open a new terminal (or run `source ~/.bashrc`) before you use mise.
Until you do, `mise doctor` reports that mise isn't activated, which is expected at this point.
For other shells, refer to [Activate mise](https://mise.jdx.dev/getting-started.html#activate-mise).

Open a new terminal so both the Homebrew and mise changes take effect,
then confirm each manager runs:

```sh
brew --version
mise --version
```

> [!NOTE]
> If you manage `~/.bashrc` with [dotfiles](./personalize-with-dotfiles.md),
> add the `brew shellenv` and `mise activate` lines to the `.bashrc` in your dotfiles repository instead,
> so applying your dotfiles doesn't overwrite them.

## Step 2: Install a tool with each manager

Install [`ripgrep`](https://github.com/BurntSushi/ripgrep) with Homebrew:

```sh
brew install ripgrep
```

Install [`bat`](https://github.com/sharkdp/bat) with mise:

```sh
mise use -g bat
```

Confirm both tools run:

```sh
rg --version
bat --version
```

Both work. So far, the two package managers look interchangeable.

## Step 3: Restart the workspace and compare

Restart the workspace.
The restart rebuilds the container from the image and keeps only your home directory.

<div class="tabs">

### UI

Open your workspace in the Coder dashboard and select **Restart**.
When it's back, reconnect your terminal: reopen the web terminal,
or run `coder ssh <your-workspace>` from your own machine.

### CLI

From a terminal on your own machine, restart the workspace by name,
then reconnect when it's back:

```sh
coder restart <your-workspace>
coder ssh <your-workspace>
```

</div>

When you reconnect, your shell prints an error before you run anything:

```text
bash: /home/linuxbrew/.linuxbrew/bin/brew: No such file or directory
```

That's the first sign something changed.
Your `.bashrc` still tries to load Homebrew, but the restart removed it.
Check each tool to see what survived.

`bat`, installed with mise, still works:

```sh
bat --version
```

```text
bat 0.26.1
```

`rg`, installed with Homebrew, is gone:

```sh
rg --version
```

```text
bash: rg: command not found
```

So is `brew` itself:

```sh
brew --version
```

```text
bash: brew: command not found
```

mise installed `bat` under `/home/coder`, which persists, so `bat` survived.
Homebrew installed `ripgrep` to `/home/linuxbrew`, outside `/home/coder`,
so the rebuild discarded Homebrew and every formula you installed with it.
The `brew shellenv` line stayed in your `.bashrc` because it lives in `/home/coder`,
which is why your shell still tries to load the missing `brew` and prints the error above.

## Step 4: Make Homebrew survive restarts

Homebrew survives a restart only if `/home/linuxbrew` survives the rebuild.
Give it a persistent volume in the template, the way the Coder dogfood template does,
so Homebrew and its formulae stay between restarts.

> [!NOTE]
> This step edits the template.
> If it isn't open for editing, refer to [Customize workspace startup](./customize-workspace-startup.md#open-the-template-for-editing).

In `main.tf`, add a volume for Homebrew's directory next to the existing `home_volume`:

```tf
resource "docker_volume" "homebrew_volume" {
  name = "coder-${data.coder_workspace.me.id}-homebrew"
  lifecycle {
    ignore_changes = all
  }
}
```

Then mount it in the `docker_container "workspace"` resource,
alongside the block that mounts `/home/coder`:

```tf
  volumes {
    container_path = "/home/linuxbrew"
    volume_name    = docker_volume.homebrew_volume.name
    read_only      = false
  }
```

Publish the change and restart the workspace:

<div class="tabs">

### UI

In the web editor, make the edits above in `main.tf`.
Select **Build**, wait for the build to pass, then select **Publish**.
On your workspace's home tab, select **Update and restart**.

### CLI

Make the edits in `~/coder-quickstart/main.tf`,
then publish and update by name:

```sh
coder templates push -d ~/coder-quickstart -y quickstart
coder update <your-workspace>
```

</div>

The restart gives you a persistent but empty `/home/linuxbrew`.
Your earlier Homebrew install is gone, so install it once more.
This time it lands on the volume:

```sh
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

The `brew shellenv` line is still in your `.bashrc` from Step 1,
so open a new terminal to load it, then reinstall `ripgrep`:

```sh
brew install ripgrep
```

Restart the workspace once more and reconnect.
This time the startup error is gone, and both tools report their versions:

```sh
rg --version
brew --version
```

Homebrew now persists because `/home/linuxbrew` lives on its own volume,
the same way mise's tools persist because they live in `/home/coder`.

## What just happened

The two package managers behaved differently for one reason: where each one installs.

- mise installs into `~/.local/share/mise`, inside your home directory, and activates from `~/.bashrc`. Both are in `/home/coder`, so its tools persist with no template change.
- Homebrew installs to `/home/linuxbrew`, outside `/home/coder`, so its tools are discarded on every restart until you mount that path on a persistent volume.

To keep a tool, choose the approach that matches who needs it:

- For a tool that's yours alone, install it with mise. It persists through restarts with no template change.
- To keep your Homebrew tools, mount `/home/linuxbrew` on a persistent volume, as you did in Step 4. This is a template change, so it affects everyone who uses the template.
- For a tool everyone needs preinstalled, add it to the startup script with `apt-get`, as in [Add a programming language](./add-a-language.md), or bake it into the workspace image.

The rule underneath all of these: a tool persists when it lives in a part of the workspace that persists.
Refer to [Resource persistence](../../admin/templates/extending-templates/resource-persistence.md) for how Coder decides what survives a restart.

## What's next?

Now that you can install your own tools, [personalize your workspace with dotfiles](./personalize-with-dotfiles.md).

## Learn more

- [Homebrew documentation](https://brew.sh/) for the package manager
- [mise documentation](https://mise.jdx.dev/) for the version manager
- [Resource persistence](../../admin/templates/extending-templates/resource-persistence.md) in the Coder documentation
- [Dotfiles](../../user-guides/workspace-dotfiles.md) in the Coder documentation
