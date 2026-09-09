import { buildModule } from "@nomicfoundation/hardhat-ignition/modules";

const DharaniModule = buildModule("DharaniModule", (m) => {
  const propertyRegistry = m.contract("PropertyRegistry");

  return { propertyRegistry };
});

export default DharaniModule;
