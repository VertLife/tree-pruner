#!/usr/bin/perl

use strict;
use DBI;

my $database = "birdtree";
my $username = "";
my $password = "";

my %frmVals = ();
my $ret = &ReadForm(\%frmVals);
my $dbh = &ConnectToPg($database, $username, $password);

print "Content-type: text/html\n\n";

&dumpData($dbh) if ( $frmVals{'password'} eq 'falcon' );

# &dumpHash();

$dbh->disconnect;
exit;


#==============================================================
sub dumpData {
	my ($dbh) = @_;

	my $statement = 'SELECT usage_date, email, number_of_trees, process, treeset_name ';
	$statement .= 'FROM usage LEFT JOIN treeset USING (treeset_id) ORDER BY usage_date ';
	my $sth = $dbh->prepare( $statement ) or die "Can't prepare $statement: $dbh->errstr\n";		
	my $rv = $sth->execute or die "can't execute the query: $sth->errstr\n";
	 
	while(my @row = $sth->fetchrow_array) {
		print "<li>$row[0]: <a href='/cgi-bin/birds/birdme.pl?file=$row[3].tre'>$row[2] trees from $row[4]</a>\n";
	}
	my $rd = $sth->finish;	
}



#==============================================================
sub ReadForm {
	my ($varRef) = @_;
	my ($i, $key, $val, $variable, @list);
	
	if (&MethGet) {
		$variable = $ENV{'QUERY_STRING'};
	} elsif (&MethPost) {
		read(STDIN, $variable, $ENV{'CONTENT_LENGTH'});
	}
	
	@list = split(/[&;]/,$variable);
	
	foreach $i (0 .. $#list) {
		$list[$i] =~ s/\+/ /g;
		($key, $val) = split(/=/,$list[$i],2);
		$key =~ s/%(..)/pack("c",hex($1))/ge;
		$val =~ s/%(..)/pack("c",hex($1))/ge;
		$$varRef{$key} .= "\0" if (defined($$varRef{$key}));
		$$varRef{$key} .= $val;
	}
	return scalar($#list + 1);
}
sub MethGet { return ($ENV{'REQUEST_METHOD'} eq "GET") };
sub MethPost { return ($ENV{'REQUEST_METHOD'} eq "POST") };

# Connect to Postgres using DBI
#==============================================================
sub ConnectToPg {
 
    my ($cstr, $user, $pass) = @_;
  
    $cstr = "DBI:Pg:dbname="."$cstr";
    $cstr .= ";host=litoria.eeb.yale.edu";
  
    my $dbh = DBI->connect($cstr, $user, $pass, {PrintError => 1, RaiseError => 1});
    $dbh || &error("DBI connect failed : ",$dbh->errstr);
 
    return($dbh);
}

# Just for testing
#==============================================================
sub dumpHash {
	foreach my $key (keys(%frmVals)) { print "$key, $frmVals{$key}; <BR>" };
}
