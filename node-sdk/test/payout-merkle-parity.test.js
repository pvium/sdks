const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const { PviumSdk } = require("../dist/index.js");

const PARITY_FIXTURE = JSON.parse(
  fs.readFileSync(
    path.join(__dirname, "../../parity-fixtures/scheduled-payout-finalize.json"),
    "utf8",
  ),
);

const BASE_PAYMENTS = [
  {
    receiver: "0x0000000000000000000000000000000000000001",
    amount: "1",
    token: "0x0000000000000000000000000000000000000002",
    decimals: 6,
    memo: "a",
  },
  {
    receiver: "0x0000000000000000000000000000000000000003",
    amount: "2",
    token: "0x0000000000000000000000000000000000000002",
    decimals: 6,
    memo: "b",
  },
  {
    receiver: "0x0000000000000000000000000000000000000004",
    amount: "3",
    token: "0x0000000000000000000000000000000000000002",
    decimals: 6,
    memo: "c",
  },
];

const EXPECTED = [
  {
    count: 1,
    merkleRoot: "0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
    proofs: [[]],
  },
  {
    count: 2,
    merkleRoot: "0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7",
    proofs: [
      ["0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b"],
      ["0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9"],
    ],
  },
  {
    count: 3,
    merkleRoot: "0xf9eb35f3a7cf94793c4f0d440f9c510828d516e6ac308141819ab7265754a8d6",
    proofs: [
      [
        "0xdeaa9357ef2c59449293ed3ba3060e6aa9cf4bf4812ecc96f2b3a6500744f05b",
        "0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
      ],
      [
        "0xc40e60ab1b114ed2eb4bbe156b73023238565bf37b45766c3c922b8089d1d2e9",
        "0x54acdb6206c4f539cbcda16b5132e71254f0ae9126b4efaa442f5c61659d544c",
      ],
      ["0x7ade83cae70b4278f73144e9a99f77f7deeed719e4778245b9f3db71f6ae02b7"],
    ],
  },
];

async function finalizedPayload(paymentCount) {
  const requests = [];
  const responses = [
    {
      meta: { statusCode: 200, success: true },
      data: {
        id: "120bdabb-5790-415c-ae75-c2fca1cc5232",
        chain: "base",
        paymentType: "Scheduled",
        complianceMode: "Open",
        metadata: {
          payoutCurrency: "0x0000000000000000000000000000000000000002",
          gracePeriod: 0,
          disapprovalDeadline: 0,
          scheduledDate: 0,
        },
        app: { clientId: "app_test" },
        payments: BASE_PAYMENTS.slice(0, paymentCount),
      },
    },
    {
      meta: { statusCode: 200, success: true },
      data: { id: "x", paymentType: "Scheduled" },
    },
  ];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  await sdk.payout.finalize(
    "120bdabb-5790-415c-ae75-c2fca1cc5232",
    {
      signerAddress: "0x0000000000000000000000000000000000000005",
      signFinalize: () => "finalize",
      signFunding: () => "funding",
    },
    { chain: "base", timestamp: 123, claimDate: 1777488000 },
  );

  return JSON.parse(requests[1].init.body);
}

test("scheduled payout merkle roots and proofs match canonical parity values", async () => {
  for (const expected of EXPECTED) {
    const payload = await finalizedPayload(expected.count);
    assert.equal(payload.merkleRoot, expected.merkleRoot);
    assert.deepEqual(
      payload.proofs.map((proof) => proof.proof),
      expected.proofs,
    );
  }
});

test("scheduled payout finalization signatures match canonical parity values", async () => {
  const requests = [];
  const responses = [
    PARITY_FIXTURE.getResponse,
    PARITY_FIXTURE.patchResponse,
  ];
  const sdk = PviumSdk.init({
    baseUrl: "http://localhost:4005/v1",
    fetchFn: async (url, init) => {
      requests.push({ url, init });
      const body = responses.shift();
      return new Response(JSON.stringify(body), {
        status: body.meta.statusCode,
        headers: { "content-type": "application/json" },
      });
    },
  });

  const result = await sdk.payout.finalize(
    PARITY_FIXTURE.payoutId,
    PARITY_FIXTURE.privateKey,
    PARITY_FIXTURE.options,
  );

  const payload = JSON.parse(requests[1].init.body);
  assert.deepEqual(payload, PARITY_FIXTURE.expectedPatchPayload);
  assert.deepEqual(result.data.payout, PARITY_FIXTURE.expectedResult.payout);
  assert.equal(result.fundingUrl, PARITY_FIXTURE.expectedResult.fundingUrl);
  assert.equal(result.data.batchDataHash, PARITY_FIXTURE.expectedResult.batchDataHash);
  assert.equal(result.data.batchHash, PARITY_FIXTURE.expectedResult.batchHash);
  assert.equal(result.data.merkleRoot, PARITY_FIXTURE.expectedResult.merkleRoot);
});
