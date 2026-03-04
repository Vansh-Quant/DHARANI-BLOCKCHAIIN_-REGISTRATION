import { buildModule } from "@nomicfoundation/hardhat-ignition/modules";

const DharaniModule = buildModule("DharaniModule", (m) => {
  const dharani = m.contract("DharaniRegistry");

  return { dharani };
});

export default DharaniModule;