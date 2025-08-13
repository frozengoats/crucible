<img width="1176" height="1176" alt="image" src="https://github.com/user-attachments/assets/d45db101-7fe8-4834-959f-dc7dc21d11dc" />
provision local and remote environments with a single binary (no dependencies)

crucible is a powerful environment configuration tool which can be used to maintain one or many identical remote environments by focussing on the creation of reusable, idempotent action sequences.  what this means is that an action sequence can be configured to execute multiple times on the same system, and only the differences should be rectified between the machine state and the desired state.

## getting started
download the latest version crucible your deployment control machine and place it in your path:
```
sudo sh -c "curl -L -o /usr/local/bin/crucible https://github.com/frozengoats/crucible/releases/latest/download/crucible && chmod +x /usr/local/bin/crucible"
```
once downloaded, run `crucible --help`

check out the [example](https://github.com/frozengoats/crucible/blob/main/testcontainer/end-to-end) project for in-depth coverage of use cases.

## project structure
projects are contained within a directory, and consist of a few components.  the core component in any project is the sequence.  a sequence is a series of steps which will be executed in the target environment.  a given project can have multple sequences, each of which perform a distinct function within the target enviornment, however the project itself encapsulates a group of environments and their configuration, both in terms of access, and the sequences which can be executed upon them.

the following is the recommended directory structure of a project:
```
my-project/       # the project directory
  resources/      # contains files and directories which may be copied or templated to the target environment(s)
  sequences/      # contains a collection of .yaml files, each of which represents an execution sequence
  config.yaml     # contains the project configuration (all the host entries and config), this file is required to be here
  values.yaml     # arbitrary, variable data available to be consumed at execution time, referenced by the sequence
```

### sequences
a sequence is a collection of actions which are executed either conditionally or unconditionally on the collection of target hosts.  think of a sequence as a series of steps involved in performing some sort of configuration or provisioning operation against one or more hosts.  an example of a sequence would be, preparing identical runtime environments on a collection of servers being managed.  an action might be to ensure that a specific piece of software is installed, or that a certain configuration entry has been made.

here's an [example](https://github.com/frozengoats/crucible/blob/main/testcontainer/end-to-end/sequences/end-to-end-test.yaml) sequence file, which is currently being used in the end-to-end test.

sequences can be reusable if desired, meaning that a sequence can effectively be parameterized, and invoked from another sequence.  this allows for development and reuse of complex patterns which could ultimately give rise to a sequence marketplace.

### configuration file<a name="config-file"></a>
the configuration file (`config.yaml`), which must reside directly within the project directory is built from the following [template](https://github.com/frozengoats/crucible/blob/main/docs/config.yaml). not all fields are required, and typically, a minimal configuration is involved, though there are many options for customization, outlined in the template.

## quick anatomy of a config file
the config file is broken into 2 main parts, the executor, and the collection of hosts.  the executor itself needs no explicit configuration, and can largely go unconfigured for most situations, however the hosts themselves must be configured in order to have a sequence executed upon them.  the basic host configuration in a config file looks like this (see config.yaml template above for details [here](#config-file)):

```
hosts:
  <host_ident>:
    host: <host>
```
this is all that's really needed for the most basic configuration.  the `host_ident` is a string key name which is used to refer to the host in invocation, debugging info, as well as config overlay files.  it serves as a memorable key name and nothing beyond that.  the `host` on the other hand, conveys the actual host and optional SSH port number (if different than 22), by which to reach the host over SSH.  this could be in the form of `<hostname>:<port>`, where hostname could be an ip address, hostname, or hostname alias (think `.ssh/config`).

## sequence anatomy
as explained above, a sequence represents a collection of individual actions which, when executed in order, make up a complete unique activity.  we will go into further detail here explaining the anatomy of the sequence, and its compositional parts.

first off, a sequence is described by a named yaml file, by convention, nested under the `sequences` directory.  this is not required, but helpful when establishing best practices.

Here is an example short sequence

```
# descriptions are not required, but greatly aid when viewing debug logs

description: prepares remote server with bare essentials

# the name is optional and serves as a location in the data context, where results accumulated through the sequence, will be stored.  this should ONLY be set when data from a sequence will be consumed externally (for instance, if the sequence is imported in another sequence)

name: prepareServer

# next is the sequence of actions which will be performed on the target host



# this action will install rsync, or do nothing if rsync is already installed
# we should always construct actions to be both idempotent and minimal, as it will ensure the most performant experience
# [sudo: true] will ensure that the command is executed as root on the remote system

- description: ensure that rsync is installed
  sudo: true
  shell: apt install rsync

# this action will run the kubectl command and generate some json output.  the `parseJson` attribute, when set to true
# will automatically interpret the stdout as json and make it available on the immediate data context as `.json`.  adding a `name`
# to the sequence, will make the output persist in the sequence-scoped context so that it can be used in another action.  in this case,
# the data will be available at `.Context.pods.json` as we will see later.

- description: collect running pods on system
  name: pods
  shell: kubectl get pods -ojson
  parseJson: true

# this action leverages the data we've already collected in the previous step and writes the length of a
# sub-item (.items) to the file `~/num_pods.txt`.

- description: write the number of pods to a file
  shell: echo {{ len(.Context.pods.json.items) }} > ~/num_pods.txt
```

## action specification
actions define units of work to perform against the remote host.  actions can be iterative or even recursive.  here is the [action specification](https://github.com/frozengoats/crucible/blob/main/docs/action.yaml) in detail.  generally speaking, most fields in actions are templatable and other parts are even exclusively comprised of evaluable expressions.  for instance, the `iterate` field in an action only takes evaluable expressions (for instance variables, or more complex values as the result of function evaluation).

consider this example action:
```
iterate: lines(.Context.files.stdout)
action:
  shell: echo {{ .item }} >> /tmp/items.txt
```

the `iterate` section of the action above only takes evaluable expressions, hence why they don't need to be wrapped in any templating markers.  the value stored on the sequence context `files.stdout`, will be passed to the `lines` function, which will split it into an array of single line strings which will then be iterated by `iterate`, causing a sub-action to be executed once per iteration.  in the sub-action, we see that the `shell` action is executed using the value `item` on the immediate context `.Values` indicates the values file context, `.Context` indicates the sequence context, and `.Host` indicates it's from the host's config context, and everything else beginnging with `.` indicates the immediate context.  in this case `iterate` will populate `item` on the immediate context, with the current item in the iteration loop.  in `shell`, which takes a string, we are using template notation `{{ <expression> }}` to indicate that we want to use the string value of `.item` as a part of the final string to be passed to the shell.

another example of this behavior is the `until` clause:

```
until:
  condition: .exitCode == 0
  maxAttempts: 3
  pauseInterval: 5
ignoreExitCode: true
shell: curl https://localhost:8888
```

in this case above, the action is NOT nested under the `until` clause.  this is because the `until` clause does not present any important new information to the action itself and thus nesting is not necessary.  in this case the `until` condition requires that the `exitCode` return a zero in order to complete.  it will retry 3 times with a 5 second delay between attempts.  we have explicitly set `ignoreExitCode` to `true` in order to disable the default behavior which is for a non-zero exit code to cause the action to fail with an error.  the curl command will execute at most 3 times if the exit code continues to be non-zero.

we can take this example one step further by adding an until clause to an iterator:

```
iterate: .Values.urls
action:
  until:
    condition: .exitCode == 0
    maxAttempts: 3
    pauseInterval: 5
  ignoreExitCode: true
  shell: curl {{ .item }}
```

in the case above, the action repeats once per url, and each url has up to 3 attempts to succeed before the entire action (top level) is deemed a failure.

## evaluable expressions
functions are implemented directly in `crucible` and a complete list can be found [here](https://github.com/frozengoats/crucible/blob/main/docs/functions.md).  they can be used in any template expression in a sequence file as well as in any evaluable expression in general.  functions can take one or more arguments but can only return single values (of any data type, meaning a single int, or a single array of n values, or a single map of n key/val pairs, etc.).

for a detailed guide on constructing valid evaluable expressions see [here](https://github.com/frozengoats/eval).

variables take the following form:
```
# variables from the values file stack take this form:
.Values.something.else[0]
.Values.key.key
.Values.myKey

# variables from the sequence context take this form:
.Context.something[-1]
.Context.key.myArray[1]
.Context.myThing.stdout

# variables from the host context take this form:
.Host.something[-1]
.Host.key.myArray[1]
.Host.myThing.stdout

# variables in the immediate action context take this form:
.item
.stdout
.exitCode
.abc123[0]

# essentially variables from the values stack start with .Values, variables in the sequence context start with .Context
# and all other action context variables start with just .
```
