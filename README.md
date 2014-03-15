SlackToGo
=========

## What does this application?

**SlackToGo** is a stand alone **Slack** integration server for distributing channels between different Slack teams in realtime.

([Stop talking... tell me how to use it!](#usage))

## What is "Slack"?

[Slack.com](https://slack.com/r/02560adf-0256965h) is a team oriented chat + fileserver.

It offers:

* Custom channels (subscription based)
* Private groups (invitation based)
* Personal messaging
* File storage for the team to share data
* Powerful search in messages and file storage
* Historical views of conversations
* Useful team statistics
* A lot of useful integrations (including GitHub!)
* Smartphone Clients
* Desktop Applications

## Why did you create SlackToGo?

I use slack primarily in two different teams. You can be member of any number of teams but Slack currently lack a feature to share one or more channels between two teams.

Because Slack offers a broad selection of API and integration functions I decided to write a "channel connector" for connect a "#friends" channel between my primary two teams which can be used to talk with the respective other team members.

## Why GoLang?

After prototyping this as PHP / Apache code I decided to switch to GoLang.

I choose Go because I could learn how to create a stand alone server executable people can use without being able to run Apache/PHP on their systems. In addition to that it is (should) be useable cross platform.

This was also done in preparation to rewrite [Hubic2SwiftGate](https://github.com/oderwat/hubic2swiftgate) in GoLang because I want to have a version for less "geeky" people to use.

With a little help of other peoples libraries I also managed to make the server easily install as a system service on Apple, Windows and Linux Systems!

Because we deal with kind of private information the gateway server also supports SSL (https).

## <a name="usage"></a>How to use it?

***I will offer binary downloads for Mac / Win / Linux at a later time!***

### Installing from source (only tested on OS X so far):

You need to have a GoLang installation!

I use [GoLang](http://golang.org/) on my mac with the help of **homebrew** but you can install go on any supported system from [here...](http://golang.org/doc/install).

With a valid go installation you can install SlackToGo with go get:

    go get github.com/oderwat/slacktogo
   
Next is changing to the source location and build it local in that directory:

    cd `go env GOPATH`/src/github.com/oderwat/slacktogo
    go build
    
After this there should be an executable named "slacktogo" in the folder. You can run it and it should complain about a missing configuration file. This will be created next!

### Modifying the config file:

In preparation to use the gateway you need to create a config file. A sample is located in the folder named `config_sample.toml`. It looks like this:

    #
    # Sample configuration for slacktogo
    #
    
    #BindAddr= # Default is to bind to all interfaces
    Port=8080
    #SSLCrt="domain_tld.crt" # I highly recommmend using SSL!
    #SSLKey="domain_tld.key"
    
    [Team.foo] # First Team
    ID="T01234FOO"
    URL="https://foo.slack.com/services/hooks/incoming-webhook?token=12345abcdeABCDE12345foo"

    [Team.bar] # Second Team
    ID="T01234BAR"
    URL="https://bar.slack.com/services/hooks/incoming-webhook?token=12345abcdeABCDE12345bar"

    [[Mapping]]
    FromTeam="foo"
    FromChannel="#friends"
    ToTeam="bar"
    ToChannel="#friends"

    [[Mapping]]
    FromTeam="bar"
    FromChannel="#friends"
    ToTeam="foo"
    ToChannel="#friends"

Create a copy of this named: `config.toml`

This sample connects two teams with the names "foo" and "bar" to share a channel named "#friends".

In the beginning you can setup the IP the server listens too (default is to listen on all interfaces. You may probably just keep it like that).

Next is the "Port" on wich the server will listen to incoming connections. It is important that this port is reachable from the slack servers so you may need to configure a port mapping at your nat-router and/or open your firewall for this port.

*I highly recommend, that you use SSL for the gateway server! I never tried without myself.*

If you don't the Outgoing WebHook will transmit unencrypted data of all conversions in the connected channels.

You need the same certificates as for any OpenSSH compatible web server (like Apache2). There can be "real" certificates or self-signed ones. There is plenty documentation how to get them.

Finally there is the part which defines your Slack teams and where you need to setup some integrations in Slack:

* Outgoing Webhooks for each Team pointing to the URL of the computer running the SlackToGo Server.
* Incoming Webhooks for each Team to be able to receive the messages from the other team(s).
* (optional) an OpenSSH compatible SSL Certficate + Key

Lets assume that you run slacktogo on a system which is reachable from the internet (by port forwarding or direct) under the subdomain `slackgate.yourdomain.tld` on port 8080.

Run the SlackToGo Server manually:

    ./slacktogo
    
or from source:

    go run slacktogo.go

You should verify that the server is reachable by opening its address in a web browser. This URL would be in this example: `http://slackgate.yourdomain.tld:8080/`. If you have a SSL Certificate (and I strongly recommend this!) you use `https://slackgate.yourdomain.tld:8080/`.

If every thing work fine you will a webpage with:

    Illegal access to Slackgate... (c) METATEXX GmbH 2014

You then need to setup an **Outgoing WebHook** in the Slack integrations panel of each team. You select the **channel** which you want to "broadcast" and leave **trigger words** empty! You may want to create a new channel for this first (e.g. #friends)!
The target URL for this WebHook is the aforementioned URL.

This hast to be done with every team and channel which gets broadcast!

After this you go to this channel in slack and write "+info" or anything. The SlackToGo Server will reply with `Unknown Team: {"T#########" "#friends"}`. Instead of `T#########` you will get the real TeamID for this team. Enter this as the ID in the config file for this team.

Do this for all the teams you want to connect!

Next you setup the **Incoming WebHooks** in the Slack integrations control panel.

Click on "EXPAND" at "Instructions for creating Incoming WebHooks" and copy the URL which is shown there. It looks similar to the URL entries in the config file. Just with your teams name and token.

Put those URLs an your real team names in the config file instead of "foo" and "bar".

Stop the SlackToGo Server (CTRL+C) and start it again.

From there on you should be able to chat with the members of the other team in the connected channel(s).

It is possible to connect "bidirectional" or to broadcast one channel to multiple different teams. This is controlled by the "Mapping" section of the config file. The syntax for this should be self explaining.

### Installing as Service

After you have the gateway working with the setup you need you can install the software as a service which auto-starts with your system:

     sudo ./slacktogo install
     sudo ./slacktogo start
     
To stop the service:

    sudo ./slacktogo stop
    
To uninstall the service:

    sudo ./slacktogo remove
    
You also can dump the configuration:

    slacktogo check
    
### SlackToGo Commands

You can check the loaded routing in the slack chat by typing:

    +routing
  
You can check the running version with

    +info

Notice: This is the first "quick" documentation for this rather complicated topic. I would love to hear about your experiences in getting this running! Feel free to write your comments into the issues section of this repository!