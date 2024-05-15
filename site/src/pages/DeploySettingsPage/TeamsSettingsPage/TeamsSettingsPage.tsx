import Button from "@mui/material/Button";
import { type FC, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "react-query";
import { createOrganization } from "api/queries/organizations";
import { myOrganizations } from "api/queries/users";
import { TextField } from "@mui/material";

const TeamsSettingsPage: FC = () => {
  const queryClient = useQueryClient();
  const addTeamMutation = useMutation(createOrganization(queryClient));
  const organizationsQuery = useQuery(myOrganizations());
  const [newOrgName, setNewOrgName] = useState("");
  return (
    <>
      <TextField
        label="New organization name"
        onChange={(event) => setNewOrgName(event.target.value)}
      />
      <p>{String(addTeamMutation.error)}</p>
      <Button onClick={() => addTeamMutation.mutate({ name: newOrgName })}>
        add new team
      </Button>

      {organizationsQuery.data?.map((org) => (
        <div key={org.id}>
          {org.name}{" "}
          <Button
            onClick={() =>
              console.log(
                "I tried to delete an org and all I got was this log message",
              )
            }
          >
            Delete
          </Button>
        </div>
      ))}
    </>
  );
};

export default TeamsSettingsPage;
