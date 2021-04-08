# End-to-End Migration Bundler Tool

Generates a variety of "accounts":

- Account with a single address containing more than the minimum migration amount
- Account with two addresses (index 0 and 100) containing together more than the minimum migration amount
- Account with the minimum migration amount spread across 100 addresses
- Account with more than the minimum migration amount spread across 100 addresses
- Account with a spent single address containing more than the minimum migration amount
- Account with the minimum migration amount spread across 100 spent addresses
- Account with more than the minimum migration amount spread across 100 spent addresses

of which their seed, bundle hash and addresses are written to a `bundles.csv`.

Example:

```
./bundlers -node="https://example.com" -seed="SEED..."
```

Note: You should compile the program with `go build -tags="pow_avx"`. MWM and output file name can also be adjusted
through CLI flags.

The provided seed must contain enough funds (~50Gi) to fund the above mentioned accounts. Use a fast computer to not fall behind
too much with used tips for the bundles with 100 addrs, as the program does not perform re-attachments.

The runtime for this program on a modern desktop computer (Ryzen 3700X) is around 12 minutes at MWM 14.

Stdout example:

```
2021/04/07 20:59:04 there are 55590605665 tokens residing on the first address of the specified seed
2021/04/07 20:59:04 generating account with one address
2021/04/07 20:59:07 generating account with minimum migration amount spread across many addresses
2021/04/07 20:59:45 generating account with random amounts spread across many addresses
2021/04/07 21:00:27 generating account with one spent address
2021/04/07 21:00:47 bundle confirmed, doing key-reuse forth-and-back
2021/04/07 21:00:49 waiting for burner bundle to be confirmed then sending back... (tail VWTXHEMOKCGGJA9KZCCRCMCNCXCALDBNRMNYDVJBXSVHOGORZNBR9HWZAQCOQNLMPLGXLRLDI9PKZ9999)
2021/04/07 21:01:07 burner bundle confirmed, sending back to origin
2021/04/07 21:01:10 generating account with minimum migration amount spread across many spent addresses
2021/04/07 21:02:23 bundle confirmed, doing key-reuse forth-and-back
2021/04/07 21:04:08 waiting for burner bundle to be confirmed then sending back... (tail QZPJE9KQYJYQFPTIZLPQPXGKXMXKEMHYEOWGXUHVFGS9GVTVDRNGLYAHMBBARSFMAHDKXPHGFMVYZ9999)
2021/04/07 21:04:27 burner bundle confirmed, sending back to origin
2021/04/07 21:06:28 generating account with random amounts spread across many spent addresses
2021/04/07 21:07:24 bundle confirmed, doing key-reuse forth-and-back
2021/04/07 21:09:29 waiting for burner bundle to be confirmed then sending back... (tail ZZOODAYOXUVXLPSWUHT9OZWJTFUGAFOIXFHNQIWNXMIIPDPXPITCJMBYEBUZGSHYWIEOXEYJUEKKA9999)
2021/04/07 21:09:50 burner bundle confirmed, sending back to origin
2021/04/07 21:11:44 done, goodbye! 12m39.705307907s
``