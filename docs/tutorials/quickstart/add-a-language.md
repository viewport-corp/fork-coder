# Add a programming language to your template

Now that you've finished [Launch your first workspace](./launch-workspace.md),
you can add another language toolchain to every workspace you create.

The Quickstart template installs a language only when the workspace owner
selects it from the **Programming Languages** parameter.
In this guide, you add Ruby as an option.

> [!NOTE]
> This guide assumes your Quickstart template is open for editing.
> If it's not, refer to [Customize workspace startup](./customize-workspace-startup.md#open-the-template-for-editing).

## What you'll do

- ✅ Add a Ruby option to the **Programming Languages** parameter.
- ✅ Publish the change and apply it to your running workspace.
- ✅ Install the Ruby toolchain so selecting Ruby works.

## Parameters in brief

A parameter is a question Coder asks when someone creates a workspace.
Each parameter comes from a `coder_parameter` [data source](https://developer.hashicorp.com/terraform/language/data-sources) in the template.
The **Programming Languages** parameter is a multi-select list,
and each language the reader can choose is an `option` block:

```tf
data "coder_parameter" "languages" {
  name         = "languages"
  display_name = "Programming Languages"
  description  = "Select the languages to pre-install in your workspace"
  type         = "list(string)"
  form_type    = "multi-select"
  default      = jsonencode(["python"])
  mutable      = true
  icon         = "/icon/code.svg"
  order        = 1

  option {
    name  = "Python"
    value = "python"
    icon  = "/icon/python.svg"
  }
  # ...more options
}
```

Adding an `option` adds a choice to that list.

## Step 1: Add the Ruby option

In `main.tf`, find the `data "coder_parameter" "languages"` block (it starts at
line 31).
The last option, **C/C++**, ends at line 71,
and the parameter's closing brace is line 72.
Add a Ruby `option` between lines 71 and 72,
after the last option and before the closing brace:

```tf
  option {
    name  = "Ruby"
    value = "ruby"
    icon  = "/icon/ruby.svg"
  }
```

> [!IMPORTANT]
> The `option` block must sit at the same indentation as the other `option`
> blocks inside the parameter.
> Coder reads the parameter's choices from these blocks,
> so a misplaced `option` doesn't appear in the form.

Now publish the change as a new template version:

<div class="tabs">

### UI

In the web editor, make the edit above in `main.tf`.
Select **Build**, wait for the build to pass, then select **Publish**.

### CLI

Make the edit in `~/coder-quickstart/main.tf`,
then publish a new version:

```sh
coder templates push -d ~/coder-quickstart -y quickstart
```

</div>

<details>
<summary>Final code: the updated parameter</summary>

The whole `languages` parameter, with Ruby added as the last option:

```tf
data "coder_parameter" "languages" {
  name         = "languages"
  display_name = "Programming Languages"
  description  = "Select the languages to pre-install in your workspace"
  type         = "list(string)"
  form_type    = "multi-select"
  default      = jsonencode(["python"])
  mutable      = true
  icon         = "/icon/code.svg"
  order        = 1

  option {
    name  = "Python"
    value = "python"
    icon  = "/icon/python.svg"
  }
  option {
    name  = "Node.js"
    value = "nodejs"
    icon  = "/icon/nodejs.svg"
  }
  option {
    name  = "Go"
    value = "go"
    icon  = "/icon/go.svg"
  }
  option {
    name  = "Rust"
    value = "rust"
    icon  = "/icon/rust.svg"
  }
  option {
    name  = "Java"
    value = "java"
    icon  = "/icon/java.svg"
  }
  option {
    name  = "C/C++"
    value = "cpp"
    icon  = "/icon/cpp.svg"
  }
  option {
    name  = "Ruby"
    value = "ruby"
    icon  = "/icon/ruby.svg"
  }
}
```

</details>

## Step 2: Add Ruby to your workspace

Your workspace from [Launch your first workspace](./launch-workspace.md) is
still on the old template version.
Update it to the version you just published,
and add Ruby to its **Programming Languages** selection:

<div class="tabs">

### UI

On your workspace, add **Ruby** to the **Programming Languages** parameter,
then select **Update and restart**.

### CLI

Update the workspace and re-select its parameters.
Replace `<your-workspace>` with your workspace's name (run `coder list` to see it):

```sh
coder update <your-workspace> --always-prompt
```

When prompted for **Programming Languages**, add **Ruby**,
then let the workspace rebuild.

</div>

## Step 3: Check whether Ruby is installed

When the workspace restarts, open a terminal and ask for the Ruby version:

```sh
ruby --version
```

The command fails:

```text
ruby: command not found
```

You added the option, selected it, and rebuilt the workspace, but Ruby isn't
there.
Adding the `option` only changed the form.
It added Ruby to the list of choices,
but nothing in the template acts on that choice yet.
A separate startup script installs each selected toolchain,
and you haven't taught it about Ruby.

## Step 4: Install Ruby when the workspace starts

The template installs each selected language from `install-languages.sh.tftpl`,
a startup script that runs when the workspace boots.
Open that file and add a branch that installs Ruby when the reader selects it:

```sh
if echo "$LANGUAGES" | grep -q "ruby"; then
  if command -v ruby >/dev/null 2>&1; then
    echo "Ruby: $(ruby --version | head -1)"
  else
    echo "Installing Ruby toolchain..."
    apt_update
    sudo apt-get install -y -qq ruby-full
    echo "Installed Ruby: $(ruby --version | head -1)"
  fi
fi
```

The script installs Ruby with `apt-get`,
the package manager built into the workspace image.

> [!WARNING]
> Use the package manager the workspace image provides, not a personal one.
> If you replace the `apt-get` line with `brew install ruby`, the build fails:
> the `codercom/enterprise-base:ubuntu` image doesn't include Homebrew,
> so the workspace logs `brew: command not found` and Ruby never installs.
> To install a personal tool like a Homebrew formula in your own workspace,
> refer to [Install your own command-line tools](./install-command-line-tools.md).

Publish the change, then update your workspace again:

<div class="tabs">

### UI

1. In the web editor, make the edit above in `install-languages.sh.tftpl`.
2. Select **Build**, wait for the build to pass, then select **Publish**.
3. On your workspace's home tab, select **Update and restart**.

### CLI

Make the edit in `~/coder-quickstart/install-languages.sh.tftpl`,
then publish and update:

```sh
coder templates push -d ~/coder-quickstart -y quickstart
coder update <your-workspace>
```

</div>

Ruby is already selected, so you don't change parameters this time.
When the workspace restarts, open a terminal and check again:

```sh
ruby --version
```

This time the workspace reports a Ruby version.

<details>
<summary>Final code: the Ruby install branch</summary>

Add this branch alongside the other language branches in
`install-languages.sh.tftpl`:

```sh
if echo "$LANGUAGES" | grep -q "ruby"; then
  if command -v ruby >/dev/null 2>&1; then
    echo "Ruby: $(ruby --version | head -1)"
  else
    echo "Installing Ruby toolchain..."
    apt_update
    sudo apt-get install -y -qq ruby-full
    echo "Installed Ruby: $(ruby --version | head -1)"
  fi
fi
```

</details>

## What just happened

You changed two different things to add one language:

- The `coder_parameter` `option` block added Ruby to the workspace creation form.
- The startup script installed the Ruby toolchain when a workspace owner selected Ruby.

A parameter collects a choice.
A startup script acts on it.
A new language needs both.

<details>
<summary>How does Coder install the language you pick?</summary>

The selection travels through the template in four steps:

1. The `option` block in `data "coder_parameter" "languages"` (`main.tf`)
   adds `ruby` to the values the form accepts.
2. When the workspace builds, `local.languages` decodes the selection
   (`main.tf` line 193):

   ```tf
   languages = jsondecode(data.coder_parameter.languages.value)
   ```

3. `coder_script.install_languages` renders the startup script with that list
   and runs it on the agent (`main.tf` lines 264-273):

   ```tf
   script = templatefile("${path.module}/install-languages.sh.tftpl", {
     LANGUAGES = join(",", local.languages)
   })
   ```

4. Inside the rendered script, the `ruby` branch matches and installs the
   toolchain:

   ```sh
   if echo "$LANGUAGES" | grep -q "ruby"; then
   ```

The first three steps ran as soon as you added the option, which is why Ruby
appeared in the form.
Step 4 is the part you were missing in Step 3,
so the script had nothing to do for `ruby`.

</details>

<details>
<summary>Why a <code>.tftpl</code> file instead of a plain script?</summary>

The install script needs to know which languages the workspace owner selected,
and only Terraform has that value when the workspace builds.
`templatefile()` renders `install-languages.sh.tftpl` and replaces
`${LANGUAGES}` with `join(",", local.languages)`,
producing a finished script with the selection baked in.
A static `.sh` file couldn't receive that value,
so the `.tftpl` is the bridge between the parameter and the shell script.

</details>

## What's next?

Now that you added a language, [install your own command-line tools](./install-command-line-tools.md).

## Learn more

- [Parameters](../../admin/templates/extending-templates/parameters.md) in the Coder documentation
- [Terraform data sources](https://developer.hashicorp.com/terraform/language/data-sources)
- [Terraform types](https://developer.hashicorp.com/terraform/language/expressions/types) for parameter values
