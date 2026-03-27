module.exports = {
  presets: [
    [
      "next/babel",
      {
        "preset-env": {
          useBuiltIns: "usage",
          corejs: 3,
          targets: {
            ie: "11",
          },
        },
      },
    ],
  ],
};
