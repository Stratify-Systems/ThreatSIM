# ThreatSIM API Endpoints

This document explains the core REST API endpoints used in ThreatSIM and how they fit together in the lifecycle of a cyber attack. Think of them as a pyramid, going from the "big picture" down to the "raw data", and then back up to the "security warnings."

## 1. `/api/v1/simulations` (The "Campaign Tracker")
**What it does:** This endpoint tells you what actual attack campaigns are currently running or have finished. 
**What it returns:** A list of "Jobs". It tells you:
* *Who* the target is (e.g., `10.0.0.1`)
* *What* type of attack is running (e.g., `brute_force` or `port_scan`)
* *Status:* Is it `RUNNING` right now, or `COMPLETED`?
**Why you use it:** If you are a team lead, you look here to say, *"Ah, Mike started a brute-force simulation against our web server 5 minutes ago."*

## 2. `/api/v1/events` (The "Raw Network Logs")
**What it does:** This is the granular, low-level telemetry. Every single tiny action a plugin takes generates an "event".
**What it returns:** A massive list of raw actions. If a brute-force attack tries 1,000 passwords, there will literally be 1,000 individual events logged here (e.g., `login_failed` on service `ssh` from `192.168.1.100`).
**Why you use it:** You look at this if you are debugging or want to see the exact raw traffic hitting the network. It's like looking at a raw firewall log.

## 3. `/api/v1/alerts` (The "Security Warnings")
**What it does:** This is the output of our **Risk Engine**. It constantly reads the raw *Events*, looks for patterns, and escalates them into readable alerts. 
**What it returns:** Highly prioritized, deduplicated warnings. Instead of showing you 1,000 failed passwords, it returns one single record that says: *"CRITICAL THREAT: IP 192.168.1.100 is conducting a Brute Force Attack."*
**Why you use it:** This is what a Security Operations Center (SOC) dashboard actually cares about. They don't want to see a million raw events; they want to see the prioritized, analyzed alerts so they know what to block.

---

### How they connect in a real scenario:
1. You start a **Simulation** against a database.
2. The Simulation fires 5,000 rapid **Events** (fake SQL injections).
3. The Risk Engine sees those events happening too fast, panics, and generates 1 `CRITICAL` **Alert**. 

When we build the dashboard, it will pull from all three of these URLs to draw a complete picture of your network health!
