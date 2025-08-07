# crucible
provision local and remote environments with a single binary (no dependencies)

crucible is a powerful environment configuration tool which can be used to maintain one or many identical remote environments by focussing on the creation of reusable, idempotent action sequences.  what this means is that an action sequence can be configured to execute multiple times on the same system, and only the differences should be rectified between the machine state and the desired state.

## getting started
download and install crucible on your deployment control machine:

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
the configuration file (`config.yaml`), which must reside directly within the project directory is built from the following [template](https://github.com/frozengoats/crucible/blob/main/template/config.yaml). not all fields are required, and typically, a minimal configuration is involved, though there are many options for customization, outlined in the template.

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