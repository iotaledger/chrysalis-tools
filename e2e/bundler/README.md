# End-to-End Migration Bundler Tool

Generates a variety of "accounts":
<table>
    <tr>
        <td><b>Scenario<b></td>
        <td><b>Description</b></td>
        <td><b>Expected Outcome</b></td>
    </tr>
    <tr>
        <td>Funds (>=1Mi) on a single unspent address (low index)</td>
        <td>Test migration with a seed with all funds on a single unspent address with index < 30.</td>
        <td>
        - Firefly should detect correct balance on the seed in the first try. User should not have to press "Check again" during the flow; 
        </br>
- Firefly should migrate user funds.
        </td>
    </tr>
   <tr>
        <td>Funds (<1Mi) on a single unspent address (low index)</td>
        <td>Test migration with a seed with all funds on a single unspent address with index < 30.</td>
        <td>
-            Firefly should detect correct balance on the seed in the first try. User should not have to press "Check again" during the flow;
</br>
- Firefly should not allow user to migrate funds.
        </td>
    </tr>
       <tr>
        <td>Funds (>=1Mi) on a single unspent address (high index)</td>
        <td>Test migration with a seed with all funds on a single unspent address with index > 30.</td>
        <td>
-       Firefly should not detect correct balance on the seed in the first try;
</br>
- Firefly should allow user to migrate funds.
        </td>
    </tr>
    <tr>
        <td>Funds (<1Mi) spread across many addresses</td>
        <td>Test migration with a seed with funds < 1Mi spread across at least 100 addresses unevenly (not in sequence but rather across random address indexes) </td>
        <td>
Firefly should not allow funds migration as accumulative balance is less the minimum migration balance (1Mi). 
        </td>
    </tr>
    <tr>
        <td>Funds (>=1Mi) spread across many addresses</td>
        <td>Test migration with a seed with funds >=1Mi spread across at least 100 addresses unevenly (not in sequence but rather across random address indexes) </td>
        <td>
Firefly should migrate all funds. 
        </td>
    </tr>
    <tr>
        <td>Mixture of funds (>=1Mi & <1Mi) spread across many unspent addresses</td>
        <td>Test migration with a seed with a mixture of funds >=1Mi & <1Mi spread across at least 100 unspent addresses unevenly</td>
        <td>
Firefly should migrate all funds. 
        </td>
    </tr>
    <tr>
        <td>Funds (>=1Mi) on a single spent address</td>
        <td>Test migration with a seed with all funds >=1Mi on a single spent address</td>
        <td>
        Firefly should correctly bundle mine the spent address and migrate the funds to the network.
        </td>
    </tr>
    <tr>
        <td>Funds (<1Mi) on a single spent address</td>
        <td>Test migration with a seed with all funds <1Mi on a single spent address</td>
        <td>
        Firefly should not bundle mine and should not allow the user to migrate funds.
        </td>
    </tr>
     <tr>
        <td>Funds (<1Mi) on multiple spent address</td>
        <td>Test migration with a seed with funds <1Mi split across multiple spent address</td>
        <td>
        Firefly should not bundle mine any address and should not allow the user to migrate funds.
        </td>
    </tr>
     <tr>
        <td>Funds (>=1Mi) on multiple spent address</td>
        <td>Test migration with a seed with funds >=1Mi split across multiple spent address</td>
        <td>
        Firefly should bundle mine all addresses and should migrate all funds to the network.
        </td>
    </tr>
    <tr>
        <td>Mixture of funds (>=1Mi & <1Mi) spread across many spent addresses</td>
        <td>Test migration with a seed with a mixture of funds >=1Mi & <1Mi spread across at least 100 spent addresses unevenly</td>
        <td>
Firefly should only bundle mine & migrate funds with >=1Mi. 
        </td>
    </tr>
       <tr>
        <td>Mixture of funds (>=1Mi & <1Mi) spread across both spent and unspent addresses</td>
        <td>Test migration with a seed with a mixture of funds >=1Mi & <1Mi spread across at least 100 spent and unspent addresses</td>
        <td>
- Firefly should bundle mine only spent addresses with >=1Mi;
</br>
- Firefly should migrate all funds from unspent addresses;
</br>
- Firefly should only migrate funds >=1Mi from spent addresses.
    </td>
    </tr>
</table>

of which their seed, bundle hash and addresses are written to a `bundles.csv`.

Example:

```
./bundler -node="https://example.com" -seed="SEED..."
```

Note: You should compile the program with `go build -tags="pow_avx"`. MWM and output file name can also be adjusted
through CLI flags.

Usage:

```
  -info-file string
        the file containing the different generated bundles (default "bundles.csv")
  -manyAddrsCount int
        the addrs count to use for scenarios which involve many addresses (default 100)
  -manyAddrsSpace int
        the index space to use for scenarios which involve many addresses (default 200)
  -manyAddrsSpentCount int
        the addrs count to use for scenarios which involve many spent addresses (default 10)
  -manyAddrsSpentSpace int
        the index space to use for scenarios which involve many spent addresses (default 30)
  -manyAddrsSpentMixedCount int
        the addrs count to use for scenarios which involve many unspent/spent addresses (default 100)
  -manyAddrsSpentMixedSpace int
        the index space to use for scenarios which involve many unspent/spent addresses (default 200)
  -mwm int
        the mwm to use for generated transactions/bundles (default 14)
  -node string
        the API URI of the node (default "https://example.com")
  -seed string
        the seed to use to fund the created bundles (default "999999999999999999999999999999999999999999999999999999999999999999999999999999999")
```

The provided seed must contain enough funds (~50Gi) to fund the above mentioned accounts. Use a fast computer to not
fall behind too much with used tips for the bundles with 100 addrs, as the program does not perform re-attachments.

The runtime for this program on a modern desktop computer (Ryzen 3700X) is around 12 minutes at MWM 14.

Stdout example:

```
2021/04/12 23:45:21 there are 55590605665 tokens residing on the first address of the specified seed
2021/04/12 23:45:21 generating scenario: Funds (>=1Mi) on a single unspent address (low index; < 30)
2021/04/12 23:45:22 done generating scenario, took 1.35213966s
2021/04/12 23:45:22 generating scenario: Funds (<1Mi) on a single unspent address (low index; < 30)
2021/04/12 23:45:23 done generating scenario, took 765.011784ms
2021/04/12 23:45:23 generating scenario: Funds (>=1Mi) on a single unspent address (high index)
2021/04/12 23:45:25 done generating scenario, took 1.764186892s
2021/04/12 23:45:25 generating scenario: Funds (<1Mi) spread across many addresses
2021/04/12 23:46:04 done generating scenario, took 38.353891452s
2021/04/12 23:46:04 generating scenario: Funds (>=1Mi) spread across many addresses
2021/04/12 23:46:39 done generating scenario, took 34.95391541s
2021/04/12 23:46:40 generating scenario: Mixture of funds (>=1Mi & <1Mi) spread across many unspent addresses
2021/04/12 23:47:18 done generating scenario, took 38.547087977s
2021/04/12 23:47:18 generating scenario: Funds (>=1Mi) on a single spent address (low index; < 30)
2021/04/12 23:47:19 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail IFLA9UOQM99V9UUYCQQUPJEOQKPASHXR9XNYJVDLGDEXWOZSOOXKCIA9CT9YRSFXLBBPUYMEENGWA9999)
2021/04/12 23:47:29 waiting for sent-off bundle to be confirmed then sending back... (tail JMV9DLK9LDC9XWRLSEKNKAXMKNBWAVSHJHYPKHDBCCCITFFIZPZZXEBIQLFVWYVO9FWIUBNFDF9GA9999)
2021/04/12 23:47:45 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:47:46 done generating scenario, took 28.223822969s
2021/04/12 23:47:46 generating scenario: Funds (<1Mi) on a single spent address (low index; < 30)
2021/04/12 23:47:47 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail NKBSBOCFNQMKNDQVBSSVHYTQCXVIG9XRPRBBIAWLCQYSICRXVWFBYGLBQTUVIZOU9UFMYQXRWYZXA9999)
2021/04/12 23:48:17 waiting for sent-off bundle to be confirmed then sending back... (tail WQIKXHOLXELWFRVYC9XUHIPAQMFXTSQJXPKGOVWSNTCLSTNSYAZZUYZLKFBWIVVOALQNJ9E9LTLZA9999)
2021/04/12 23:48:40 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:48:40 done generating scenario, took 53.651071635s
2021/04/12 23:48:40 generating scenario: Funds (<1Mi) spread across many spent addresses
2021/04/12 23:48:45 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail AZLIKXI9GLBAOURDIMEPRSOQDUYUJYKNFTISOBDNYKFBXFTOPASYPWPEYOPCAIYDBMQBSALCYFYW99999)
2021/04/12 23:49:06 waiting for sent-off bundle to be confirmed then sending back... (tail NVZUTNNR9XOLBRQYRCQGZCQDPULXVFDQYWWDRVLRALGUPCYUFMOOQOJLNPMZECQXHKEMIOKQHVE9A9999)
2021/04/12 23:49:26 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:49:39 done generating scenario, took 58.356541525s
2021/04/12 23:49:39 generating scenario: Funds (>=1Mi) spread across many spent addresses
2021/04/12 23:49:42 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail KTCLVXLBGPEJU9AYEXE9BDHGX9HJJDWUBWJKVV9DXARRAKGIKSFG9GLPGCYWSDOHVFEQIDZSAUOE99999)
2021/04/12 23:50:11 waiting for sent-off bundle to be confirmed then sending back... (tail IOBSZEPDFKYJOJLUEFIPTMWEVRISVCOKQOOACJKQBUVKJXFWHFYQAKLKUSTLUAIJYUISQCQNDKRYA9999)
2021/04/12 23:50:26 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:50:36 done generating scenario, took 56.941681452s
2021/04/12 23:50:36 generating scenario: Mixture of funds (>=1Mi & <1Mi) spread across many spent addresses
2021/04/12 23:51:14 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail MDLUYQZKDJQLCHSFAKQJEHWSPFNCXESECJRGDENRUQHQQGQZTXYBFPSN9RNMRPELIEPZGSA9EBWEZ9999)
2021/04/12 23:53:26 waiting for sent-off bundle to be confirmed then sending back... (tail GZATBAGHTGH9IJZUCXJNVTHPGZHKRQ9M9NHMJXUVPMXJAMVGUDUTJRR9LAADGY99KKTBQMQPKLBR99999)
2021/04/12 23:53:43 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:55:44 done generating scenario, took 5m7.943599393s
2021/04/12 23:55:44 generating scenario: Mixture of funds (>=1Mi & <1Mi) spread across both spent and unspent addresses
2021/04/12 23:56:27 waiting for scenario bundle to be confirmed before sending forth/back to spent addrs... (tail KXIXWFNEB9AVAGCYEWJQAUBYQELMVDZP9GTAZKBU9MIM9P9GVQYNHUCR9TCHVFYU9YGBLUEHFHVR99999)
2021/04/12 23:57:26 waiting for sent-off bundle to be confirmed then sending back... (tail GJMUPBNLORWULFAXXLTWGRLOBAHAEOPS9YYECXWLPQZJRZESPDCIFIEOUQIIATAGQUWBFLNRGLIXZ9999)
2021/04/12 23:57:48 sent-off bundle confirmed, sending back to spent addresses...
2021/04/12 23:58:31 done generating scenario, took 2m46.129551702s
2021/04/12 23:58:31 done, goodbye! 13m9.483080135s

``
