# Welcome to Issue Agent

## Introduction

Issue Agent is a Large Language Model(LLM) Based Agent.

When an Agent is given an issue, it will autonomously attempt to solve it and submit the results as a Pull Request on GitHub.


### Very limited scope
Issue Agent is very limited in what it can do.  This is because it is designed to be secure and usable in practice.

What we mean by secure here are the following:

* Do not execute the response returned from the LLM as is. For example, executing shell. Only predefined functions can be executed.
* Control the credentials given to the Agent. 


### What the Issue Agent can and cannot do

* ✅ Pull requests only be created in the repository configured in the configuration file (Setup)
* ✅ To read an issue in one GitHub repository and submit a PR to that repository
* 🚫 Interactive development work between an Agent and the human who directs the Agent
* 🚫 Commits or pull request to an unconfigured repository
* 🚫 Free operation using shell commands such as find, curl, etc.


### ...

Concept and details will be explained in concept page.
