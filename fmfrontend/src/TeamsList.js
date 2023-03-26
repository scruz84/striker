import React, { Component } from 'react';
import {Link} from "react-router-dom";
import {Button, ButtonGroup, Container, Table} from "reactstrap";
import AppNavbar from "./components/AppNavbar";

class TeamsList extends Component {

    constructor(props) {
        super(props);
        this.state = {teams: []};
        //this.remove = this.remove.bind(this);
    }

    componentDidMount() {
        fetch('/v1/teams')
            .then(response => response.json())
            .then(data => this.setState({teams: data}));
    }

    render() {
        const {teams} = this.state;

        const teamList = teams.map(team => {
            return <tr key={team.Id}>
                <td style={{whiteSpace: 'nowrap'}}>{team.name}</td>
                <td>{team.name}</td>
                <td>
                    <ButtonGroup>
                        <Button size="sm" color="primary" tag={Link} to={"/teams/" + team.id}>Edit</Button>
                        <Button size="sm" color="danger" onClick={() => this.remove(team.Id)}>Delete</Button>
                    </ButtonGroup>
                </td>
            </tr>
        });

        return (
            <div>
                <AppNavbar/>
                <Container fluid>
                    <div className="float-right">
                        <Button color="success" tag={Link} to="/teams/new">Add Client</Button>
                    </div>
                    <h3>teams</h3>
                    <Table className="mt-4">
                        <thead>
                        <tr>
                            <th width="30%">Name</th>
                            <th width="30%">Email</th>
                            <th width="40%">Actions</th>
                        </tr>
                        </thead>
                        <tbody>
                        {teamList}
                        </tbody>
                    </Table>
                </Container>
            </div>
        );
    }
}

export default TeamsList;