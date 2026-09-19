import { notFound } from "next/navigation";
import { Workspace } from "@/components/Workspace";
import { getSystemById } from "@/lib/systems";

type PageProps = {
  params: Promise<{ id: string }>;
};

export default async function SystemPage({ params }: PageProps) {
  const { id } = await params;
  const system = getSystemById(id);
  if (!system) {
    notFound();
  }
  return <Workspace system={system} />;
}
