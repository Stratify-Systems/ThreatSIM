# This is ThreatSim.

Hello, world! 

If you've ever built a web application, you know that the moment you put it on the internet, someone, somewhere, is going to try to break it. They're going to type `' OR 1=1--` into your login box. They're going to put `<script>alert(1)</script>` into your search bar. They are going to try to bypass your authentication and steal your users' data.

And so, as developers, we have a problem. How do we find these vulnerabilities *before* the bad guys do?

We *could* just sit at our keyboards, opening our web browsers, and typing these malicious inputs into every single input field, one by one. But humans are slow. Humans make mistakes. As computer scientists, we look at that manual, repetitive, tedious process and we ask ourselves: **Can we automate this?**

Can we write a program whose sole purpose is to test the security of *other* programs?

Enter **ThreatSim**.

## The Abstraction

At its core, ThreatSim is a security validation engine. But we didn't want you to have to write hundreds of lines of Go or Python just to test a simple API endpoint. We wanted to create an **abstraction**. 

Instead of writing code, what if you could just *declare* what you want to happen?

```yaml
  - name: "Admin Login Bruteforce Test"
    plugin: "bruteforce"
    config:
      path: "/login"
      username: "admin"
```

Look at how simple that is! In just six lines of YAML, we have completely abstracted away the complexity of network protocols, TCP handshakes, HTTP headers, memory management, and loop mechanics. You are simply declaring to the computer: *"I want you to bruteforce the `/login` path using the username `admin`."*

## The Engine: Under the Hood

But how does the computer actually *do* this? How do we go from six lines of text on a hard drive to a barrage of HTTP requests over the network?

If we look under the hood of ThreatSim, we see a beautiful application of **Separation of Concerns**. 

We have the **CLI Layer**—the part of the program that talks to *you*, the human. It reads your commands, parses your flags, and loads that YAML file into memory.

But the CLI doesn't actually execute the attacks. No, it hands that data structure off to the **Execution Engine**. 

The Engine is the brain of ThreatSim. It iterates over your simulations. And when it sees that you've requested a `plugin` like `bruteforce`, it does something incredibly powerful.

## The Plugin Architecture: Polymorphism in Action

Instead of hardcoding every possible attack directly into the Engine (which would make our codebase massive and impossible to maintain), we've designed ThreatSim using a **Plugin Architecture**. 

We defined a contract—an `interface` in Go. We said to the compiler, "As long as a piece of code has a `Name()` and can `Execute()`, we will treat it as a plugin."

When the Engine sees `plugin: "bruteforce"`, it goes to its internal registry, finds the Bruteforce plugin, and says, *"Here is the target URL, and here is the configuration the user provided. Do your worst."*

The plugin then takes over. The Bruteforce plugin doesn't just send one request. It looks at the parameters you provided. It pulls a dictionary of common passwords. And then, using the sheer speed of a compiled Go binary, it iterates through every single password, formulating dozens of unique HTTP requests in mere milliseconds.

It sends them to the server, and it listens. It looks at the response. Did the server return a `200 OK` for an invalid password? If so, the plugin catches it, flags the expected security behavior as violated, and reports back to the Engine: **Vulnerability Found.**

## The Result

And finally, the Engine takes all of those results, wraps them up, and hands them to the Reporting module. The Reporting module applies ANSI color codes and prints a beautiful, human-readable summary right back to your terminal screen. 

Green checkmarks for safe endpoints. Red crosses for vulnerabilities.

From a simple text file, to a complex, automated orchestration of HTTP requests, to a secure application.

This is automation. This is policy-as-code. 

This is **ThreatSim**.
